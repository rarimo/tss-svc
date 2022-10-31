package sign

import (
	"context"
	"encoding/binary"
	goerr "errors"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/session"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign/step"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

// Service implements singleton pattern
var service *Service

var (
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

// Service implements the full flow of the threshold signing of proposed pool.
// During receiving new blocks notifications in the NewBlock method service will run the flow steps if possible.
// The tss flow consists of the following steps:
// 1. Proposing: the derived proposer proposes the next pool of operations to sign
// 2. Accepting: all parties shares their acceptances to start signing the pool.
// 3. Signing
type Service struct {
	mu sync.Mutex

	params *local.Params
	secret *local.Secret
	con    *connectors.BroadcastConnector
	conf   *connectors.ConfirmConnector
	pool   *pool.Pool

	step          *step.Step
	session       session.ISession
	lastSignature string

	cancelCtx   context.CancelFunc
	controllers map[types.StepType]step.IController

	rarimo  *grpc.ClientConn
	log     *logan.Entry
	storage *pg.Storage
}

// NewService returns new Service but only once because Service implements the singleton pattern for simple usage as
// the same instance in all injections.
// The first session information will be fetched from the service configuration file and the previous session
// will be mocked to wait for the first one.
func NewService(cfg config.Config) *Service {
	if service == nil {
		service = &Service{
			params:        local.NewParams(cfg),
			secret:        local.NewSecret(cfg),
			con:           connectors.NewBroadcastConnector(cfg),
			conf:          connectors.NewConfirmConnector(cfg),
			pool:          pool.NewPool(cfg),
			step:          step.NewLastStep(cfg.Session().StartBlock - 1),
			session:       session.NewDefaultSession(cfg.Session().StartSessionId-1, cfg.Session().StartBlock-1),
			lastSignature: cfg.Session().LastSignature,
			controllers:   make(map[types.StepType]step.IController),
			rarimo:        cfg.Cosmos(),
			log:           cfg.Log(),
			storage:       cfg.Storage(),
		}

		service.log.Infof("--- Next session on block: %d with id: %d ---", service.session.End()+1, service.session.ID()+1)
	}
	return service
}

// NewBlock receives new blocks from timer
func (s *Service) NewBlock(height uint64) error {
	s.log.Infof("--- New block: %d ---", height)
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session.IsFinished(height) {
		s.log.Infof("--- Session %d finished---", s.session.ID())
		s.stopController()

		if ok := s.session.FinishSign(); ok {
			s.log.Infof("--- Session %d Successful! ---", s.session.ID())
			s.finish()
		} else {
			s.log.Infof("failed to finish signing step")
			s.fail()
		}

		s.nextSession()
		return nil
	}

	if s.step.Next(height) {
		s.log.Infof("--- Step finished. Next step: %s ---", s.step.Type().String())
		s.stopController()

		switch s.step.Type() {
		case types.StepType_Accepting:
			if ok := s.session.FinishProposal(); !ok {
				s.log.Infof("failed to finish proposal step")
				s.fail()
			}
		case types.StepType_Signing:
			if ok := s.session.FinishAcceptance(); !ok {
				s.log.Infof("failed to finish acceptance step")
				s.fail()
			}
		}

		s.nextStep()
	}

	return nil
}

// Receive method receives the new MsgSubmitRequest from the parties and routes them to the corresponding controller.
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
	s.log.Infof("Scheduling next session id=%d", s.session.ID()+1)
	s.params.UpdateParams()
	s.controllers = make(map[types.StepType]step.IController)

	proposer := s.getProposer(s.session.ID() + 1)
	s.log.Infof("Proposer account: %s", proposer.Account)
	s.log.Debugf("Proposer pub key: %s", proposer.PubKey)
	s.step = step.NewStep(s.params, s.session.End()+1)

	s.session = session.NewSession(
		s.session.ID()+1,
		s.session.End()+1,
		s.step.EndAllBlock(),
		proposer,
		s.storage,
	)

	s.nextStep()
}

func (s *Service) nextStep() {
	s.controllers[s.step.Type()] = s.getStepController()
	if s.session.IsProcessing() {
		s.log.Infof("Running controller for step: %s", s.step.Type().String())
		var ctx context.Context
		ctx, s.cancelCtx = context.WithCancel(context.Background())
		s.controllers[s.step.Type()].Run(ctx)
	}
}

func (s *Service) fail() {
	s.session.Fail()
	for _, index := range s.session.Indexes() {
		err := s.pool.Add(index)
		if err != nil {
			s.log.WithError(err).Error("failed adding back operation to the pool")
		}
	}
}

func (s *Service) finish() {
	if len(s.session.Indexes()) > 0 {
		if err := s.conf.SubmitConfirmation(s.session.Indexes(), s.session.Root(), s.session.Signature()); err != nil {
			s.log.WithError(err).Debug("error submitting confirmation. maybe already submitted")
		}
		// TODO fix unstable
		s.lastSignature = s.session.Signature()
	}
}

func (s *Service) stopController() {
	if s.cancelCtx != nil {
		s.cancelCtx()
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
			s.secret,
			s.session.GetSignatureChanel(),
			s.log,
		)
	}

	return nil
}

func (s *Service) getProposer(sessionId uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(s.lastSignature)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, sessionId)
	hash := crypto.Keccak256(sigBytes, idBytes)
	return *s.params.Parties()[int(hash[len(hash)-1])%s.params.N()]
}
