package sign

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign/controllers"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type Session struct {
	log    *logan.Entry
	mu     *sync.Mutex
	id     uint64
	bounds *core.BoundsManager

	factory   *controllers.ControllerFactory
	current   controllers.IController
	isStarted bool
	cancel    context.CancelFunc
}

var _ core.ISession = &Session{}

func NewSession(cfg config.Config) *Session {
	factory := &controllers.ControllerFactory{}

	return &Session{
		mu:      &sync.Mutex{},
		log:     cfg.Log(),
		id:      cfg.Session().StartSessionId,
		bounds:  &core.BoundsManager{},
		factory: factory,
		current: factory.GetProposalController(),
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

	if s.bounds.SessionEnd <= height {
		s.stopController()
		return
	}

	if s.current != nil {
		if !s.isStarted {
			s.runController()
		}

		if s.bounds.GetBounds(s.current.Type()).End <= height {
			s.stopController()
			s.current = s.current.Next()
			s.isStarted = false
			s.runController()
		}
	}
}

func (s *Session) NextSession() core.ISession {
	factory := s.factory.NextFactory()
	return &Session{
		mu:  &sync.Mutex{},
		log: s.log,
		id:  s.id + 1,
		// TODO
		bounds:  &core.BoundsManager{},
		factory: factory,
		current: factory.GetProposalController(),
	}
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
	}
}

func (s *Session) stopController() {
	if s.current != nil {
		s.cancel()
		s.current.WaitFor()
	}
}
