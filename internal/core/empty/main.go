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

func NewEmptySession(cfg config.Config, sessionType types.SessionType, creator func(cfg config.Config) core.ISession) *Session {
	if timer.GetTimer().CurrentBlock() < cfg.Session().StartBlock {
		return &Session{
			nextF: func() core.ISession {
				return creator(cfg)
			},
			end: cfg.Session().StartBlock - 1,
			log: cfg.Log(),
		}
	}

	// Changing config to start next session
	cfg.Session().StartSessionId = GetSessionId(cfg.Session().StartSessionId, cfg.Session().StartBlock, sessionType) + 1
	cfg.Session().StartBlock = GetSessionEnd(cfg.Session().StartSessionId, cfg.Session().StartBlock, sessionType) + 1

	return &Session{
		nextF: func() core.ISession {
			return creator(cfg)
		},
		end: cfg.Session().StartBlock - 1,
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
