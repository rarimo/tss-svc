package sign

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/controllers"
	"gitlab.com/rarimo/tss/tss-svc/internal/data"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// Session represents default and reshare sessions that is a normal flow
type Session struct {
	log    *logan.Entry
	mu     sync.Mutex
	id     uint64
	bounds *core.BoundsManager

	factory   *controllers.ControllerFactory
	current   controllers.IController
	isStarted bool
	cancel    context.CancelFunc
	data      *pg.Storage
}

// Implements core.ISession interface
var _ core.ISession = &Session{}

func NewSession(cfg config.Config, id, startBlock uint64) core.ISession {
	factory := controllers.NewControllerFactory(cfg, id, types.SessionType_DefaultSession)
	next := &Session{
		log:     cfg.Log().WithField("id", id).WithField("type", types.SessionType_DefaultSession.String()),
		id:      id,
		bounds:  core.NewBoundsManager(startBlock, types.SessionType_DefaultSession),
		factory: factory,
		data:    cfg.Storage(),
		current: factory.GetProposalController(),
	}
	next.initSessionData()
	return next
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) Receive(request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil {
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

	s.log.Infof("[Sign session] Running next block %d on session #%d", height, s.id)

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
		}
	}
}

func (s *Session) NextSession() core.ISession {
	factory := s.factory.NextFactory(types.SessionType_DefaultSession)
	next := &Session{
		log:     s.log.WithField("id", s.id+1).WithField("type", types.SessionType_DefaultSession.String()),
		id:      s.id + 1,
		bounds:  core.NewBoundsManager(s.End()+1, types.SessionType_DefaultSession),
		factory: factory,
		data:    s.data,
		current: factory.GetProposalController(),
	}
	next.initSessionData()
	return next
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
	if s.current != nil && s.cancel != nil {
		s.cancel()
		s.current.WaitFor()
	}
}

func (s *Session) initSessionData() {
	err := s.data.DefaultSessionDatumQ().Insert(&data.DefaultSessionDatum{
		ID:         int64(s.id),
		Status:     int(types.SessionStatus_SessionProcessing),
		BeginBlock: int64(s.bounds.SessionStart),
		EndBlock:   int64(s.bounds.SessionEnd),
	})

	if err != nil {
		s.log.WithError(err).Error("Error creating session entry")
	}
}
