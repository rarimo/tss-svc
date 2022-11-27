package core

import (
	goerr "errors"
	"sync"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrInvalidSessionID = goerr.New("invalid session ID")
)

type ISession interface {
	ID() uint64
	End() uint64
	Receive(request *types.MsgSubmitRequest) error
	NewBlock(height uint64)
	NextSession() ISession
}

type SessionManager struct {
	mu             *sync.Mutex
	currentSession ISession
}

func (s *SessionManager) Receive(request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.Id != s.currentSession.ID() {
		return ErrInvalidSessionID
	}

	return s.currentSession.Receive(request)
}

func (s *SessionManager) NewBlock(height uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentSession.NewBlock(height)
	if s.currentSession.End() <= height {
		s.currentSession = s.currentSession.NextSession()
	}
}
