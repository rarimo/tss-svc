package keygen

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/controllers"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/sign"
	"gitlab.com/rarimo/tss/tss-svc/internal/data"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// Session represents key generation session that should be run only once as a first session
type Session struct {
	log    *logan.Entry
	mu     sync.Mutex
	id     uint64
	bounds *core.BoundsManager

	factory   *controllers.ControllerFactory
	current   controllers.IController
	data      *pg.Storage
	isStarted bool
	cancel    context.CancelFunc
}

// Implements core.ISession interface
var _ core.ISession = &Session{}

func NewSession(cfg config.Config) core.ISession {
	factory := controllers.NewControllerFactory(cfg)

	sess := &Session{
		log:     cfg.Log(),
		id:      cfg.Session().StartSessionId,
		bounds:  core.NewBoundsManager(cfg.Session().StartBlock),
		factory: factory,
		data:    cfg.Storage(),
		current: factory.GetKeygenController(),
	}
	sess.initSessionData()
	return sess
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

	s.log.Infof("[Session] Running next block %d on keygen session #%d", height, s.id)

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
	return sign.NewSessionWithData(s.id+1, s.End()+1, s.factory.NextFactory(), s.data, s.log)
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

func (s *Session) initSessionData() {
	session, err := s.data.SessionQ().SessionByID(int64(s.id), false)
	if err != nil {
		s.log.WithError(err).Error("Error selecting session")
		return
	}

	if session == nil {
		err := s.data.SessionQ().Insert(&data.Session{
			ID:         int64(s.id),
			Status:     int(types.SessionStatus_SessionProcessing),
			BeginBlock: int64(s.bounds.SessionStart),
			EndBlock:   int64(s.bounds.SessionEnd),
		})

		if err != nil {
			s.log.WithError(err).Error("Error creating session entry")
		}
	}
}
