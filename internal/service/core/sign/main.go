package sign

import (
	"context"
	goerr "errors"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/step"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const (
	MaxPoolSize        = 32
	StepProposingIndex = 0
	StepAcceptingIndex = 1
	StepSigningIndex   = 2
)

var (
	ErrUnsupportedContent = goerr.New("unsupported content")
	ErrInvalidRequestType = goerr.New("invalid request type")
	ErrSignerNotAParty    = goerr.New("signer not a party")
	ErrInvalidSignature   = goerr.New("invalid signature")
	ErrProcessingRequest  = goerr.New("error processing request")

	stepForRequest = map[types.RequestType]types.StepType{
		types.RequestType_Proposal:   types.StepType_Proposing,
		types.RequestType_Acceptance: types.StepType_Accepting,
		types.RequestType_Sign:       types.StepType_Signing,
	}
)

type Service struct {
	mu sync.Mutex

	params *local.Params
	secret *local.Secret
	con    *connectors.BroadcastConnector
	pool   *pool.Pool

	step    *step.Step
	session *session.Session

	cancelCtx   context.CancelFunc
	controllers map[types.StepType]step.IController

	rarimo  *grpc.ClientConn
	log     *logan.Entry
	storage *pg.Storage
}

// NewBlock receives new blocks from timer
func (s *Service) NewBlock(height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session.IsFinished(height) {
		s.cancelCtx()

		if ok := s.session.FinishSign(); !ok {
			s.session.Fail()
		}

		s.nextSession()
		return nil
	}

	if s.step.Next(height) {
		s.cancelCtx()

		if ok := s.session.FinishProposal(); !ok {
			s.session.Fail()
		}

		s.nextStep()
	}

	return nil
}

func (s *Service) Receive(request types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.step.Type() != stepForRequest[request.Type] {
		return ErrInvalidRequestType
	}

	sender, err := s.AuthRequest(request)
	if err != nil {
		return err
	}

	if err = s.controllers[s.step.Type()].Receive(sender, request); err != nil {
		s.log.WithError(err).Debug("failed to process request")
		return ErrProcessingRequest
	}

	return nil
}

func (s *Service) AuthRequest(request types.MsgSubmitRequest) (rarimo.Party, error) {
	hash := crypto.Keccak256(request.Details.Value)

	signature, err := hexutil.Decode(request.Signature)
	if err != nil {
		s.log.WithError(err).Debug("failed to decode signature")
		return rarimo.Party{}, ErrInvalidSignature
	}

	pub, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		s.log.WithError(err).Debug("failed to recover signature pub key")
		return rarimo.Party{}, ErrInvalidSignature
	}

	party, ok := s.params.Party(hexutil.Encode(pub))
	if !ok {
		return rarimo.Party{}, ErrSignerNotAParty
	}

	return party, nil
}

func (s *Service) nextSession() {
	s.params.UpdateParams()
	s.controllers = make(map[types.StepType]step.IController)

	// TODO calculate
	var proposer rarimo.Party

	s.session = session.NewSession(
		s.session.ID()+1,
		s.session.End()+1,
		s.params,
		proposer,
		s.storage,
	)

	s.step = step.NewStep(s.params, s.session.Start())
	s.nextStep()
}

func (s *Service) nextStep() {
	s.controllers[s.step.Type()] = s.getStepController()
	if !s.session.IsFailed() {
		var ctx context.Context
		ctx, s.cancelCtx = context.WithCancel(context.Background())
		s.controllers[s.step.Type()].Run(ctx)
	}
}

func (s *Service) getStepController() step.IController {
	switch s.step.Type() {
	case types.StepType_Proposing:
		return step.NewProposalController(
			s.session.ID(),
			s.params,
			s.secret,
			s.session.Proposer(),
			s.session.GetProposalChanel(),
			s.con,
			s.pool,
			s.rarimo,
			s.log,
		)
	case types.StepType_Accepting:
		return step.NewAcceptanceController(
			s.session.Root(),
			s.session.GetAcceptanceChanel(),
			s.con,
			s.params,
			s.log,
		)
	case types.StepType_Signing:
		return step.NewSignatureController(
			s.session.Root(),
			s.params,
			s.session.GetSignatureChanel(),
			s.log,
		)
	}

	return nil
}

/*func (p *ProposalController) nextProposer(signature string, nextSessionId uint64) *rarimo.Party {
	sigBytes := hexutil.MustDecode(signature)
	stepBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(stepBytes, nextSessionId)
	hash := crypto.Keccak256(sigBytes, stepBytes)
	return p.tssP.Parties[int(hash[len(hash)-1])%len(p.tssP.Parties)]
}
*/
