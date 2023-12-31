package reshare

import (
	"context"
	"sync"

	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/internal/core/controllers"
	"github.com/rarimo/tss-svc/internal/data"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3"
)

// Session represents reshare session that performs key-regeneration after updating parties set
type Session struct {
	log    *logan.Entry
	mu     sync.Mutex
	id     uint64
	bounds *core.BoundsManager

	data      *controllers.LocalSessionData
	current   controllers.IController
	isStarted bool
	cancel    context.CancelFunc
}

// Implements core.ISession interface
var _ core.ISession = &Session{}

func NewSession(ctx core.Context, id, startBlock uint64) core.ISession {
	data := controllers.NewSessionData(ctx, id, types.SessionType_ReshareSession)

	sess := &Session{
		log:     ctx.Log().WithField("id", id).WithField("type", types.SessionType_ReshareSession.String()),
		id:      id,
		bounds:  core.NewBoundsManager(startBlock, types.SessionType_ReshareSession),
		data:    data,
		current: data.GetProposalController(),
	}
	sess.initSessionData(ctx)
	return sess
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) Receive(ctx context.Context, request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current != nil {
		return s.current.Receive(core.GetSessionCtx(ctx, types.SessionType_ReshareSession), request)
	}

	return nil
}

func (s *Session) NewBlock(height uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bounds.SessionStart > height {
		return
	}

	s.log.Infof("[Reshare session] Running next block %d on session #%d", height, s.id)

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
	data := s.data.Next()
	next := &Session{
		log:     s.log.WithField("id", s.id+1).WithField("type", types.SessionType_ReshareSession.String()),
		id:      s.id + 1,
		bounds:  core.NewBoundsManager(s.End()+1, types.SessionType_ReshareSession),
		data:    data,
		current: data.GetProposalController(),
	}

	next.initSessionData(core.DefaultSessionContext(types.SessionType_ReshareSession))
	return next
}

func (s *Session) End() uint64 {
	return s.bounds.SessionEnd
}

func (s *Session) runController() {
	if s.current != nil {
		var ctx context.Context
		ctx, s.cancel = context.WithCancel(core.GetSessionCtx(context.TODO(), types.SessionType_ReshareSession))
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

func (s *Session) initSessionData(ctx core.Context) {
	err := ctx.PG().ReshareSessionDatumQ().Insert(&data.ReshareSessionDatum{
		ID:         int64(s.id),
		Status:     int(types.SessionStatus_SessionProcessing),
		BeginBlock: int64(s.bounds.SessionStart),
		EndBlock:   int64(s.bounds.SessionEnd),
	})

	if err != nil {
		s.log.WithError(err).Error("Error creating session entry")
	}
}
