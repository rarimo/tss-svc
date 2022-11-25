package reshare

import (
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type Session struct {
}

var _ core.ISession = &Session{}

func (s *Session) ID() uint64 {
	//TODO implement me
	panic("implement me")
}

func (s *Session) Bounds() *core.Bounds {
	//TODO implement me
	panic("implement me")
}

func (s *Session) Receive(request *types.MsgSubmitRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s *Session) NewBlock(height uint64) {
	//TODO implement me
	panic("implement me")
}

func (s *Session) NextSession() core.ISession {
	//TODO implement me
	panic("implement me")
}
