package core

import (
	goerr "errors"
	"sync"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrInvalidSessionID = goerr.New("invalid session ID")
)

// ISession represents session component that is responsible for launching first session controller,
// managing controllers execution and creating next sessions.
type ISession interface {
	ID() uint64
	End() uint64
	Receive(request *types.MsgSubmitRequest) error
	// NewBlock is a receiver for timer.Timer
	NewBlock(height uint64)
	NextSession() ISession
}

// SessionManager is responsible for managing session execution
type SessionManager struct {
	mu             sync.Mutex
	currentSession ISession
}

func NewSessionManager(session ISession) *SessionManager {
	return &SessionManager{
		currentSession: session,
	}
}

func (s *SessionManager) Receive(request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if request.Id != s.currentSession.ID() {
		return ErrInvalidSessionID
	}

	return s.currentSession.Receive(request)
}

func (s *SessionManager) NewBlock(height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentSession.NewBlock(height)
	if s.currentSession.End() <= height {
		s.currentSession = s.currentSession.NextSession()
	}
	return nil
}

func (s *SessionManager) ID() uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.currentSession.ID()
}
