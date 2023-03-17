package empty

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/timer"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

type Session struct {
	nextF func() core.ISession
	end   uint64
	log   *logan.Entry
}

func NewEmptySession(cfg config.Config, sessionType types.SessionType, creator func(cfg config.Config, id, startBlock uint64) core.ISession) *Session {
	currentId := cfg.Session().StartSessionId - 1
	endBlock := cfg.Session().StartBlock - 1

	defer func() {
		cfg.Log().Infof("[Empty Session] Running empty session for type=%s", sessionType.String())
		cfg.Log().Infof("[Empty Session] ID = %d End = %d", currentId, endBlock)
	}()

	if timer.GetTimer().CurrentBlock() >= cfg.Session().StartBlock {
		currentId = GetSessionId(cfg.Session().StartSessionId, cfg.Session().StartBlock, sessionType)
		endBlock = GetSessionEnd(currentId, cfg.Session().StartBlock, sessionType)
	}

	return &Session{
		nextF: func() core.ISession {
			return creator(cfg, currentId+1, endBlock+1)
		},
		end: endBlock,
		log: cfg.Log(),
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

func (s *Session) Receive(*types.MsgSubmitRequest) error {
	return nil
}

func (s *Session) NewBlock(height uint64) {
	s.log.Infof("[Empty session] Running next block %d", height)
}

func (s *Session) NextSession() core.ISession {
	return s.nextF()
}
