package keygen

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/controllers"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type Session struct {
	log    *logan.Entry
	mu     sync.Mutex
	id     uint64
	bounds *core.BoundsManager

	factory   *controllers.ControllerFactory
	current   controllers.IController
	isStarted bool
	cancel    context.CancelFunc
}

var _ core.ISession = &Session{}

func NewSession(cfg config.Config) *Session {
	factory := controllers.NewControllerFactory(cfg)

	return &Session{
		log:     cfg.Log(),
		id:      cfg.Session().StartSessionId,
		bounds:  core.NewBoundsManager(cfg.Session().StartBlock),
		factory: factory,
		current: factory.GetKeygenController(),
	}
}

func (s *Session) ID() uint64 {
	return s.id
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

	if s.bounds.SessionStart > height {
		return
	}

	s.log.Infof("Running next block on keygen session #%d", s.id)

	if s.bounds.SessionEnd <= height {
		s.stopController()
		return
	}

	if s.current != nil {
		if !s.isStarted {
			s.runController()
		}

		if s.bounds.Current().End <= height {
			s.stopController()
			s.current = s.current.Next()
			s.isStarted = false
			s.runController()
		}
	}
}

func (s *Session) NextSession() core.ISession {
	factory := s.factory.NextFactory()
	return sign.NewSessionWithData(s.id+1, s.End()+1, factory, s.log)
}

func (s *Session) End() uint64 {
	return s.bounds.SessionEnd
}

func (s *Session) runController() {
	if s.current != nil {
		var ctx context.Context
		ctx, s.cancel = context.WithCancel(context.TODO())
		s.current.Run(ctx)
		s.isStarted = true
		s.bounds.NextController(s.current.Type())
	}
}

func (s *Session) stopController() {
	if s.current != nil {
		s.cancel()
		s.current.WaitFor()
	}
}
