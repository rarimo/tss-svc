package sign

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign/controllers"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

type Session struct {
	log    *logan.Entry
	mu     *sync.Mutex
	id     uint64
	bounds *core.Bounds

	client    *grpc.ClientConn
	factory   *controllers.ControllerFactory
	current   controllers.IController
	isStarted bool
	cancel    context.CancelFunc
}

var _ core.ISession = &Session{}

func NewSession(cfg config.Config) *Session {
	params, err := core.NewParamsSnapshot(cfg.Cosmos())
	if err != nil {
		panic(err)
	}

	factory := controllers.NewControllerFactory(
		connectors.NewCoreConnector(cfg),
		secret.NewLocalStorage(cfg),
		params,
		cfg.Cosmos(),
		pool.NewPool(cfg),
		cfg.Log(),
		core.NewProposer(cfg).WithParams(params),
	)

	if err != nil {
		return &Session{
			mu:     &sync.Mutex{},
			log:    cfg.Log(),
			id:     cfg.Session().StartSessionId,
			client: cfg.Cosmos(),
			// TODO
			factory: factory,
			bounds:  core.NewBounds(cfg.Session().StartBlock, 0),
		}
	}

	return &Session{
		mu:     &sync.Mutex{},
		log:    cfg.Log(),
		id:     cfg.Session().StartSessionId,
		client: cfg.Cosmos(),
		// TODO
		bounds:  core.NewBounds(cfg.Session().StartBlock, 0),
		factory: factory,
		current: factory.GetProposalController(cfg.Session().StartSessionId, core.NewBounds(cfg.Session().StartBlock, params.Step(controllers.ProposingIndex).Duration)),
	}
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) Bounds() *core.Bounds {
	return s.bounds
}

func (s *Session) Receive(request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil && request.Id == s.id {
		return s.current.Receive(request)
	}

	return nil
}

func (s *Session) NewBlock(height uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bounds.Finish <= height {
		s.stopController()
		return
	}

	if s.current != nil {
		if !s.isStarted {
			s.runController()
		}

		if s.current.Bounds().Finish <= height {
			s.stopController()
			s.current = s.current.Next()
			s.isStarted = false
			s.runController()
		}
	}
}

func (s *Session) NextSession() core.ISession {
	params, err := core.NewParamsSnapshot(s.client)
	factory := s.factory.NewWithParams(params)
	if err != nil {
		return &Session{
			mu:     &sync.Mutex{},
			log:    s.log,
			id:     s.id + 1,
			client: s.client,
			// TODO
			factory: factory,
			bounds:  core.NewBounds(s.bounds.Finish+1, 0),
		}
	}

	// TODO check params active

	return &Session{
		mu:     &sync.Mutex{},
		log:    s.log,
		id:     s.id + 1,
		client: s.client,
		// TODO
		bounds:  core.NewBounds(s.bounds.Finish+1, 0),
		factory: factory,
		current: factory.GetProposalController(s.id+1, core.NewBounds(s.bounds.Finish+1, params.Step(controllers.ProposingIndex).Duration)),
	}
}

func (s *Session) runController() {
	if s.current != nil {
		var ctx context.Context
		ctx, s.cancel = context.WithCancel(context.TODO())
		s.current.Run(ctx)
		s.isStarted = true
	}
}

func (s *Session) stopController() {
	if s.current != nil {
		s.cancel()
		s.current.WaitFor()
	}
}
