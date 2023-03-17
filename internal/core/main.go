package core

import (
	goerr "errors"
	"sync"

	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

var (
	ErrInvalidSessionID   = goerr.New("invalid session ID")
	ErrInvalidSessionType = goerr.New("invalid session type")
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
	mu       sync.Mutex
	sessions map[types.SessionType]ISession
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[types.SessionType]ISession),
	}
}

func (s *SessionManager) AddSession(sessionType types.SessionType, session ISession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[sessionType] = session
}

func (s *SessionManager) Receive(request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[request.SessionType]; ok && session != nil {
		if session.ID() == request.Id {
			return session.Receive(request)
		}

		return ErrInvalidSessionID
	}

	return ErrInvalidSessionType
}

func (s *SessionManager) NewBlock(height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for sessionType, session := range s.sessions {
		if session != nil {
			session.NewBlock(height)
			if session.End() <= height {
				s.sessions[sessionType] = session.NextSession()
			}
		}
	}

	return nil
}

func (s *SessionManager) ID(sessionType types.SessionType) (uint64, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[sessionType]; ok && session != nil {
		return session.ID(), true
	}

	return 0, false
}
