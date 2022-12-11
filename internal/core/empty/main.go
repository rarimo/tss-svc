package empty

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type Session struct {
	nextF func() core.ISession
	end   uint64
	log   *logan.Entry
}

func NewEmptySession(cfg config.Config, creator func(cfg config.Config) core.ISession) *Session {
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
