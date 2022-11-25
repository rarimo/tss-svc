package core

import (
	goerr "errors"
	"sync"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrInvalidSessionID = goerr.New("invalid session ID")
)

type Bounds struct {
	Start  uint64
	Finish uint64
}

func NewBounds(start, duration uint64) *Bounds {
	return &Bounds{
		Start:  start,
		Finish: start + duration,
	}
}

type ISession interface {
	ID() uint64
	Bounds() *Bounds
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
	if s.currentSession.Bounds().Finish <= height {
		s.currentSession = s.currentSession.NextSession()
	}
}
