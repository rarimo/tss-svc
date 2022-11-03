package keygen

import (
	"context"
	goerr "errors"
	"math/big"
	"sync"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	cosmostypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/auth"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrInvalidRequestType = goerr.New("invalid request type")
	ErrProcessingRequest  = goerr.New("error processing request")
)

type Service struct {
	*connectors.BroadcastConnector
	*auth.RequestAuthorizer
	mu sync.Mutex

	receiver *core.Receiver
	params   *local.Params
	secret   *local.Secret

	party tss.Party
	log   *logan.Entry
}

func init() {
	tss.SetCurve(secp256k1.S256())
}

func NewService(cfg config.Config) *Service {
	return &Service{
		BroadcastConnector: connectors.NewBroadcastConnector(cfg),
		RequestAuthorizer:  auth.NewRequestAuthorizer(cfg),
		receiver:           core.NewReceiver(local.NewParams(cfg).N()),
		params:             local.NewParams(cfg),
		secret:             local.NewSecret(cfg),
		log:                cfg.Log(),
	}
}

var _ core.IGlobalReceiver = &Service{}

func (s *Service) Run() {
	parties := tss.SortPartyIDs(s.params.PartyIds())
	peerCtx := tss.NewPeerContext(parties)
	params := tss.NewParameters(peerCtx, s.secret.PartyId(), len(parties), s.params.T())

	out := make(chan tss.Message)
	end := make(chan keygen.LocalPartySaveData)

	s.party = keygen.NewLocalParty(params, out, end)

	go func() {
		err := s.party.Start()
		if err != nil {
			panic(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.TODO())
	go s.listenOutput(ctx, out)

	result := <-end
	cancel()

	s.log.Infof("Pub key: %s", hexutil.Encode(result.ECDSAPub.Bytes()))
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

			toParties := s.params.PartyIds()
			if !msg.IsBroadcast() {
				toParties = msg.GetTo()
			}

			for _, to := range toParties {
				party, _ := s.params.PartyByAccount(to.Id)
				for {
					if _, err := s.Submit(ctx, party, request); err == nil {
						break
					}
				}
			}
		}
	}
}

func (s *Service) Receive(request *types.MsgSubmitRequest) error {
	sender, err := s.Auth(request)
	if err != nil {
		return err
	}
	s.receiver.Receive(sender, request)
	return nil
}

// Receive method receives the new MsgSubmitRequest from the parties.
func (s *Service) receive() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for {
		msg, ok := <-s.receiver.Order
		if !ok {
			break
		}

		if msg.Request.Type == types.RequestType_Keygen {
			partyId := tss.NewPartyID(msg.Sender.Account, "", new(big.Int).SetBytes(hexutil.MustDecode(msg.Sender.PubKey)))
			_, err := s.party.UpdateFromBytes(msg.Request.Details.Value, partyId, true)
			if err != nil {
				s.log.WithError(err).Error("error updating party")
			}
		}
	}
}
