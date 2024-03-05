package core

import (
	"context"
	goerr "errors"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"sync"

	"github.com/rarimo/tss-svc/pkg/types"
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
	Receive(ctx context.Context, request *types.MsgSubmitRequest) error
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

func (s *SessionManager) Receive(ctx context.Context, request *types.MsgSubmitRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session, ok := s.sessions[request.Data.SessionType]; ok && session != nil {
		if session.ID() == request.Data.Id {
			return session.Receive(ctx, request)
		}

		return errors.From(ErrInvalidSessionID, logan.F{
			"current_id":  session.ID(),
			"received_id": request.Data.Id,
		})
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
