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

func NewEmptySession(cfg config.Config, creator func(cfg config.Config) core.ISession) *Session {
	current := timer.NewTimer(cfg).CurrentBlock()
	if current < cfg.Session().StartBlock {
		return &Session{
			nextF: func() core.ISession {
				return creator(cfg)
			},
			end: cfg.Session().StartBlock - 1,
			log: cfg.Log(),
		}
	}

	// For core.SessionDuration = 23 blocks
	// 10-33 34-57 58-81, start = 10 block

	// id = (current - start) / 24 + 1
	// current = 10 => id = (10 - 10) / 24 + 1 = 1
	// current = 33 => id = (33 - 10) / 24 + 1 = 1
	// current = 34 => id = (34 - 10) / 24 + 1 = 1 + 1 = 1
	sessionId := (current-cfg.Session().StartBlock)/(core.SessionDuration+1) + cfg.Session().StartSessionId

	// end = id*24 + start - 1
	// id = 1 => end = 1 * 24 + 10 - 1 = 33
	// id = 2 => end = 2 * 24 + 10 - 1 = 57
	// id = 3 => end = 3 * 24 + 10 - 1 = 81
	sessionEnd := sessionId*(core.SessionDuration+1) + cfg.Session().StartBlock - 1

	// Changing config to start next session
	cfg.Session().StartSessionId = sessionId
	cfg.Session().StartBlock = sessionEnd + 1

	return &Session{
		nextF: func() core.ISession {
			return creator(cfg)
		},
		end: sessionEnd,
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
