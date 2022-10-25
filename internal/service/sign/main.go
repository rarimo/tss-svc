package sign

import (
	"context"
	goerr "errors"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors/party"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/step"
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
	ErrInvalidStep        = goerr.New("invalid session step")
	ErrSignerNotAParty    = goerr.New("signer not a party")
	ErrInvalidSignature   = goerr.New("invalid signature")
	ErrProcessingRequest  = goerr.New("error processing request")
)

type Service struct {
	mu sync.Mutex

	pool *pool.Pool
	con  *party.SubmitConnector

	params *local.Storage

	step      *step.Step
	session   *session.Session
	ctx       context.Context
	cancelCtx context.CancelFunc

	proposal   *step.ProposalController
	acceptance *step.AcceptanceController
	signature  *step.SignatureController

	log     *logan.Entry
	rarimo  *grpc.ClientConn
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
			return nil
		}

		switch s.step.Type() {
		case types.StepType_Accepting:
			s.acceptance = step.NewAcceptanceController(
				s.session.Root(),
				s.session.GetAcceptanceChanel(),
				s.params,
				s.log,
			)
			s.ctx, s.cancelCtx = context.WithCancel(context.Background())
			s.acceptance.Run(s.ctx)
		case types.StepType_Signing:
			s.signature = step.NewSignatureController(
				s.session.Root(),
				s.params,
				s.session.GetSignatureChanel(),
				s.log,
			)
			s.ctx, s.cancelCtx = context.WithCancel(context.Background())
			s.signature.Run(s.ctx)

		}
	}

	return nil
}

func (s *Service) ReceiveProposal(request types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.Type != types.RequestType_Proposal {
		return ErrInvalidRequestType
	}

	if s.step.Type() != types.StepType_Proposing {
		return ErrInvalidStep
	}

	sender, err := s.AuthRequest(request)
	if err != nil {
		s.log.WithError(err).Debug("failed to recover signature pub key")
		return ErrInvalidSignature
	}

	if sender == nil {
		return ErrSignerNotAParty
	}

	if err = s.proposal.ReceiveProposal(sender, request); err != nil {
		s.log.WithError(err).Debug("failed to process request")
		return ErrProcessingRequest
	}

	return nil
}

func (s *Service) ReceiveAcceptance(request types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.Type != types.RequestType_Acceptance {
		return ErrInvalidRequestType
	}

	if s.step.Type() != types.StepType_Accepting {
		return ErrInvalidStep
	}

	sender, err := s.AuthRequest(request)
	if err != nil {
		s.log.WithError(err).Debug("failed to recover signature pub key")
		return ErrInvalidSignature
	}

	if sender == nil {
		return ErrSignerNotAParty
	}

	if err = s.acceptance.ReceiveAcceptance(sender, request); err != nil {
		s.log.WithError(err).Debug("failed to process request")
		return ErrProcessingRequest
	}

	return nil
}

func (s *Service) ReceiveSign(request types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.Type != types.RequestType_Sign {
		return ErrInvalidRequestType
	}

	if s.step.Type() != types.StepType_Signing {
		return ErrInvalidStep
	}

	sender, err := s.AuthRequest(request)
	if err != nil {
		s.log.WithError(err).Debug("failed to recover signature pub key")
		return ErrInvalidSignature
	}

	if sender == nil {
		return ErrSignerNotAParty
	}

	if err = s.signature.ReceiveSign(sender, request); err != nil {
		s.log.WithError(err).Debug("failed to process request")
		return ErrProcessingRequest
	}

	return nil
}

func (s *Service) AuthRequest(request types.MsgSubmitRequest) (*rarimo.Party, error) {
	hash := crypto.Keccak256(request.Details.Value)

	signature, err := hexutil.Decode(request.Signature)
	if err != nil {
		return nil, err
	}

	pub, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		return nil, err
	}

	return s.params.Party(hexutil.Encode(pub)), nil
}

func (s *Service) nextSession() {
	s.params.UpdateParams()

	// TODO calculate
	var proposer *rarimo.Party

	s.session = session.NewSession(
		s.session.ID()+1,
		s.session.End()+1,
		s.params,
		proposer,
		s.storage,
	)

	s.step = step.NewStep(s.params, s.session.Start())

	s.proposal = step.NewProposalController(
		s.session.ID(),
		s.params,
		proposer,
		s.session.GetProposalChanel(),
		s.pool,
		s.rarimo,
		s.log,
	)

	s.acceptance = nil
	s.signature = nil

	s.ctx, s.cancelCtx = context.WithCancel(context.Background())
	s.proposal.Run(s.ctx)
}

/*func (p *ProposalController) nextProposer(signature string, nextSessionId uint64) *rarimo.Party {
	sigBytes := hexutil.MustDecode(signature)
	stepBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(stepBytes, nextSessionId)
	hash := crypto.Keccak256(sigBytes, stepBytes)
	return p.tssP.Parties[int(hash[len(hash)-1])%len(p.tssP.Parties)]
}
*/
