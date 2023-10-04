package empty

import (
	"context"

	"github.com/rarimo/tss-svc/internal/config"
	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3"
)

type Session struct {
	nextF func() core.ISession
	end   uint64
	log   *logan.Entry
}

func NewEmptySession(ctx core.Context, info *config.SessionInfo, sessionType types.SessionType, creator func(ctx core.Context, id, startBlock uint64) core.ISession) *Session {
	currentId := info.StartSessionId - 1
	endBlock := info.StartBlock - 1

	defer func() {
		ctx.Log().Infof("[Empty Session] Running empty session for type=%s", sessionType.String())
		ctx.Log().Infof("[Empty Session] ID = %d End = %d", currentId, endBlock)
	}()

	if current := ctx.Timer().CurrentBlock(); current >= info.StartBlock {
		currentId = GetSessionId(current, info.StartSessionId, info.StartBlock, sessionType)
		endBlock = GetSessionEnd(currentId, info.StartBlock, sessionType)
	}

	return &Session{
		nextF: func() core.ISession {
			return creator(ctx, currentId+1, endBlock+1)
		},
		end: endBlock,
		log: ctx.Log(),
	}
}

// Implements core.ISession interface
var _ core.ISession = &Session{}

func (s *Session) ID() uint64 {
	return 0
}

func (s *Session) End() uint64 {
	return s.end
}

func (s *Session) Receive(context.Context, *types.MsgSubmitRequest) error {
	return nil
}

func (s *Session) NewBlock(height uint64) {
	s.log.Infof("[Empty session] Running next block %d", height)
}

func (s *Session) NextSession() core.ISession {
	return s.nextF()
}
