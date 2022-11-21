package core

import (
	"context"
	"crypto/elliptic"
	"fmt"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/ecdsa/resharing"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3/errors"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ReshareController struct {
	*defaultController
	*bounds
	mu *sync.Mutex
	wg *sync.WaitGroup

	sessionId uint64
	data      types.AcceptanceData
	parties   tss.SortedPartyIDs
	end       chan keygen.LocalPartySaveData
	out       chan tss.Message
	party     tss.Party

	index  map[string]uint
	status bool

	factory *ControllerFactory
}

func NewReshareController(
	sessionId uint64,
	data types.AcceptanceData,
	defaultController *defaultController,
	bounds *bounds,
	factory *ControllerFactory,
) IController {
	return &ReshareController{
		defaultController: defaultController,
		bounds:            bounds,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		sessionId:         sessionId,
		data:              data,
		parties:           defaultController.params.PartyIds(),
		end:               make(chan keygen.LocalPartySaveData, 1),
		out:               make(chan tss.Message, 1000),
		index:             make(map[string]uint),
		factory:           factory,
	}
}

var _ IController = &ReshareController{}

func (r *ReshareController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := r.auth.Auth(request)
	if err != nil {
		return err
	}

	return r.receive(sender, request)
}

func (r *ReshareController) receive(sender rarimo.Party, request *types.MsgSubmitRequest) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if request.Type == types.RequestType_Reshare {
		r.infof("Received message from %s", sender.Account)
		r.index[sender.Account]++
		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := r.party.UpdateFromBytes(request.Details.Value, r.parties.FindByKey(new(big.Int).SetBytes(data)), request.IsBroadcast)
		if err != nil {
			return errors.Wrap(err, "error updating party")
		}
	}
	return nil
}

func (r *ReshareController) Run(ctx context.Context) {
	localId := r.parties.FindByKey(r.secret.PartyId().KeyInt())

	key, err := r.secret.GetLocalPartyData()
	if err != nil {
		empty := keygen.NewLocalPartySaveData(r.reshare.OldN())
		key = &empty
		key.LocalPreParams = *r.secret.MustGetLocalPartyPreParams()
	}

	params := tss.NewReSharingParameters(s256k1.S256(), tss.NewPeerContext(r.reshare.OldSet()), tss.NewPeerContext(r.parties), localId, r.reshare.OldN(), r.reshare.OldN(), r.reshare.NewN(), r.reshare.NewT())
	r.party = resharing.NewLocalParty(params, *key, r.out, r.end)

	go func() {
		err := r.party.Start()
		if err != nil {
			panic(err)
		}
	}()

	r.wg.Add(2)
	go r.run(ctx)
	go r.listenOutput(ctx)
}

func (r *ReshareController) run(ctx context.Context) {
	defer func() {
		r.infof("Controller finished")
		r.wg.Done()
	}()

	<-ctx.Done()

	select {
	case result, ok := <-r.end:
		if !ok {
			r.errorf(nil, "TSS Party chanel closed")
			return
		}

		r.infof("Pub key: %s", hexutil.Encode(elliptic.Marshal(s256k1.S256(), result.ECDSAPub.X(), result.ECDSAPub.Y())))
		r.secret.UpdateLocalPartyData(&result)
		r.reshare.Complete(result.Ks, result.BigXj, result.ECDSAPub)
		r.status = true
	default:
		r.infof("Reshare process has not been finished yet or has some errors")
	}
}

func (r *ReshareController) WaitFor() {
	defer r.wg.Wait()
}

func (r *ReshareController) Next() IController {
	if r.status {
		sBounds := NewBounds(r.End()+1, r.params.Step(SigningIndex).Duration)
		return r.factory.GetSignatureController(r.sessionId, r.data, sBounds)
	}

	bounds := NewBounds(
		r.End()+1,
		r.params.Step(SigningIndex).Duration+
			1+r.params.Step(FinishingIndex).Duration,
	)

	return r.factory.GetFinishController(r.sessionId, types.SignatureData{}, bounds)
}

func (r *ReshareController) listenOutput(ctx context.Context) {
	defer r.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-r.out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				r.errorf(err, "failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Reshare,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			r.infof("Sending to %v", msg.GetTo())
			for _, to := range msg.GetTo() {
				r.infof("Sending message to %s", to.Id)
				party, _ := r.params.PartyByAccount(to.Id)

				if party.Account == r.secret.AccountAddressStr() {
					r.infof("Sending to self")
					r.receive(party, request)
					continue
				}

				r.MustSubmitTo(ctx, request, &party)
			}
		}
	}
}

func (r *ReshareController) infof(msg string, args ...interface{}) {
	r.Infof("[Reshare] - %s", fmt.Sprintf(msg, args))
}

func (r *ReshareController) errorf(err error, msg string, args ...interface{}) {
	r.WithError(err).Errorf("[Reshare] - %s", fmt.Sprintf(msg, args))
}
