package step

import (
	"context"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gogo/protobuf/proto"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type SignatureController struct {
	*core.Receiver
	wg   *sync.WaitGroup
	id   uint64
	root string

	result chan *session.Signature

	params *local.Params
	secret *local.Secret

	log *logan.Entry
}

func NewSignatureController(
	id uint64,
	root string,
	params *local.Params,
	secret *local.Secret,
	result chan *session.Signature,
	log *logan.Entry,
) *SignatureController {
	return &SignatureController{
		Receiver: core.NewReceiver(params.N()),
		wg:       &sync.WaitGroup{},
		id:       id,
		root:     root,
		params:   params,
		secret:   secret,
		result:   result,
		log:      log,
	}
}

var _ IController = &SignatureController{}

func (s *SignatureController) receive() {
	for {
		msg, ok := <-s.Order
		if !ok {
			break
		}

		if msg.Request.Type == types.RequestType_Sign {
			sign := new(types.SignRequest)

			if err := proto.Unmarshal(msg.Request.Details.Value, sign); err != nil {
				s.log.WithError(err).Error("error unmarshalling request")
			}

			if sign.Root == s.root {
				// TODO
			}
		}
	}
}

func (s *SignatureController) Run(ctx context.Context) {
	s.wg.Add(1)
	go s.run(ctx)
}

// TODO mocked for one party
func (s *SignatureController) run(ctx context.Context) {
	signature, err := crypto.Sign(hexutil.MustDecode(s.root), s.secret.ECDSAPrvKey())
	if err != nil {
		s.log.WithError(err).Error("error signing root hash")
		return
	}

	s.log.Infof("[Signing %d] - Signed root %s signature %s", s.id, s.root, hexutil.Encode(signature))

	s.result <- &session.Signature{
		Signed:    []string{s.secret.ECDSAPubKeyStr()},
		Signature: hexutil.Encode(signature),
	}

	s.log.Infof("[Signing %d] - Controller finished", s.id)
	s.wg.Done()
	close(s.Order)
}

func (s *SignatureController) WaitFinish() {
	s.wg.Wait()
}
