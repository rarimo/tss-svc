package keygen

import (
	"context"
	"crypto/elliptic"
	"encoding/json"
	"math/big"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	s256k1 "github.com/btcsuite/btcd/btcec"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/auth"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// Service implements singleton pattern
var service *Service

// Service implements the full flow of the threshold key generation.
type Service struct {
	*connectors.BroadcastConnector
	*auth.RequestAuthorizer
	*core.RequestQueue
	mu sync.Mutex

	parties tss.SortedPartyIDs
	localId *tss.PartyID
	params  *local.Params
	secret  *local.Secret

	party tss.Party
	log   *logan.Entry
}

func NewService(cfg config.Config) *Service {
	if service == nil {
		service = &Service{
			BroadcastConnector: connectors.NewBroadcastConnector(cfg),
			RequestAuthorizer:  auth.NewRequestAuthorizer(cfg),
			RequestQueue:       core.NewQueue(local.NewParams(cfg).N()),
			params:             local.NewParams(cfg),
			secret:             local.NewSecret(cfg),
			log:                cfg.Log(),
		}
	}

	return service
}

var _ core.IGlobalReceiver = &Service{}
var _ core.IReceiver = &Service{}

func (s *Service) Run() {
	s.parties = s.params.PartyIds()
	s.localId = s.parties.FindByKey(s.secret.PartyId().KeyInt())
	peerCtx := tss.NewPeerContext(s.parties)
	params := tss.NewParameters(s256k1.S256(), peerCtx, s.localId, len(s.parties), s.params.T())

	out := make(chan tss.Message, 1000)
	end := make(chan keygen.LocalPartySaveData, 1000)

	s.party = keygen.NewLocalParty(params, out, end, *s.secret.GetLocalPartyPreParams())

	go func() {
		err := s.party.Start()
		if err != nil {
			panic(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.TODO())
	go s.ProcessQueue(ctx, s)
	go s.listenOutput(ctx, out)

	result := <-end
	cancel()

	data, _ := json.Marshal(result)
	s.log.Info(string(data))

	s.log.Infof("Pub key: %s", hexutil.Encode(elliptic.Marshal(s256k1.S256(), result.ECDSAPub.X(), result.ECDSAPub.Y())))
	s.log.Infof("Xi: %s", hexutil.Encode(result.Xi.Bytes()))
	s.log.Infof("Ki: %s", hexutil.Encode(result.ShareID.Bytes()))
}

func (s *Service) listenOutput(ctx context.Context, out <-chan tss.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-out:
			details, err := cosmostypes.NewAnyWithValue(msg.WireMsg().Message)
			if err != nil {
				s.log.WithError(err).Error("failed to parse details")
				continue
			}

			request := &types.MsgSubmitRequest{
				Type:        types.RequestType_Keygen,
				IsBroadcast: msg.IsBroadcast(),
				Details:     details,
			}

			toParties := msg.GetTo()
			if msg.IsBroadcast() {
				toParties = s.params.PartyIds()
			}

			s.log.Infof("Sending to %v", toParties)
			for _, to := range toParties {
				s.log.Infof("Sending message to %s", to.Id)
				party, _ := s.params.PartyByAccount(to.Id)

				if party.Account == s.secret.AccountAddressStr() {
					s.log.Info("Sending to self")
					s.ReceiveFromSender(party, request)
					continue
				}

				s.MustSubmitTo(ctx, request, &party)
			}
		}
	}
}

func (s *Service) Receive(request *types.MsgSubmitRequest) error {
	sender, err := s.Auth(request)
	if err != nil {
		return err
	}

	s.Queue <- &core.Msg{
		Request: request,
		Sender:  sender,
	}
	return nil
}

func (s *Service) ReceiveFromSender(sender rarimo.Party, request *types.MsgSubmitRequest) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.Type == types.RequestType_Keygen {
		s.log.Infof("Received message from %s", sender.Account)

		_, data, _ := bech32.DecodeAndConvert(sender.Account)
		_, err := s.party.UpdateFromBytes(request.Details.Value, s.parties.FindByKey(new(big.Int).SetBytes(data)), request.IsBroadcast)
		if err != nil {
			s.log.WithError(err).Error("error updating party")
		}
	}
}
