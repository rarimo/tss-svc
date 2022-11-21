package core

import (
	"context"
	"crypto/elliptic"
	"fmt"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3/errors"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type KeygenController struct {
	*defaultController
	*bounds

	mu *sync.Mutex
	wg *sync.WaitGroup

	parties tss.SortedPartyIDs
	end     chan keygen.LocalPartySaveData
	out     chan tss.Message
	party   tss.Party

	status bool

	factory *ControllerFactory
}

func NewKeygenController(
	defaultController *defaultController,
	bounds *bounds,
	factory *ControllerFactory,
) IController {
	return &KeygenController{
		defaultController: defaultController,
		bounds:            bounds,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		parties:           defaultController.params.PartyIds(),
		end:               make(chan keygen.LocalPartySaveData, 1),
		out:               make(chan tss.Message, 1000),
		factory:           factory,
	}
}

var _ IController = &KeygenController{}

func (k *KeygenController) Receive(request *types.MsgSubmitRequest) error {
	sender, err := k.auth.Auth(request)
	if err != nil {
		return err
	}

	return k.receive(sender, request)
}

func (k *KeygenController) receive(sender rarimo.Party, request *types.MsgSubmitRequest) error {
	k.mu.Lock()
	defer k.mu.Unlock()

	if request.Type == types.RequestType_Keygen {
		k.infof("Received message from %s", sender.Account)
		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := k.party.UpdateFromBytes(request.Details.Value, k.parties.FindByKey(new(big.Int).SetBytes(data)), request.IsBroadcast)
		if err != nil {
			return errors.Wrap(err, "error updating party")
		}
	}
	return nil
}

func (k *KeygenController) Run(ctx context.Context) {
	peerCtx := tss.NewPeerContext(k.parties)
	localId := k.parties.FindByKey(k.secret.PartyId().KeyInt())
	params := tss.NewParameters(s256k1.S256(), peerCtx, localId, k.params.N(), k.params.T())

	k.party = keygen.NewLocalParty(params, k.out, k.end, *k.secret.MustGetLocalPartyPreParams())
	go func() {
		err := k.party.Start()
		if err != nil {
			k.errorf(err, "error starting tss party")
			close(k.end)
		}
	}()

	k.wg.Add(2)
	go k.run(ctx)
	go k.listenOutput(ctx)
}

func (k *KeygenController) run(ctx context.Context) {
	defer func() {
		k.infof("Controller finished")
		k.wg.Done()
	}()

	<-ctx.Done()

	select {
	case result, ok := <-k.end:
		if !ok {
			k.errorf(nil, "TSS Party chanel closed")
			return
		}

		k.infof("Pub key: %s", hexutil.Encode(elliptic.Marshal(s256k1.S256(), result.ECDSAPub.X(), result.ECDSAPub.Y())))
		k.secret.UpdateLocalPartyData(&result)
		k.status = true
	default:
		k.infof("Keygen process has not been finished yet or has some errors")
	}
}

func (k *KeygenController) listenOutput(ctx context.Context) {
	defer k.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-k.out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				k.errorf(err, "failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Keygen,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			k.infof("Sending to %v", msg.GetTo())
			for _, to := range msg.GetTo() {
				k.infof("Sending message to %s", to.Id)
				party, _ := k.params.PartyByAccount(to.Id)

				if party.Account == k.secret.AccountAddressStr() {
					k.infof("Sending to self")
					if err := k.receive(party, request); err != nil {
						k.errorf(err, "Failed to update self")
					}
					continue
				}

				k.MustSubmitTo(ctx, request, &party)
			}
		}
	}
}

func (k *KeygenController) WaitFor() {
	k.wg.Wait()
}

func (k *KeygenController) Next() IController {
	if k.status {
		return k.factory.GetFinishController(1, types.SignatureData{}, NewBounds(k.finish+1, k.params.Step(FinishingIndex).Duration))
	}
	panic("failed to process keygen")
}

func (k *KeygenController) infof(msg string, args ...interface{}) {
	k.Infof("[Keygen] - %s", fmt.Sprintf(msg, args))
}

func (k *KeygenController) errorf(err error, msg string, args ...interface{}) {
	k.WithError(err).Errorf("[Keygen] - %s", fmt.Sprintf(msg, args))
}
