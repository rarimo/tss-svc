package old

import (
	"database/sql"
	"fmt"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/data"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type Session struct {
	*logan.Entry

	id          uint64
	initialized bool
	storage     *pg.Storage
}

func NewSession(cfg config.Config) *Session {
	return &Session{
		Entry:   cfg.Log(),
		storage: cfg.Storage(),
		id:      cfg.Session().StartSessionId - 1,
	}
}

func (s *Session) SessionID() uint64 {
	return s.id
}

func (s *Session) NextSession(id uint64, proposer string, bounds *bounds) {
	s.id = id
	s.initialized = true
	err := s.storage.SessionQ().Upsert(&data.Session{
		ID:     int64(s.id),
		Status: int(types.Status_Processing),
		Proposer: sql.NullString{
			String: proposer,
			Valid:  true,
		},
		BeginBlock: int64(bounds.start),
		EndBlock:   int64(bounds.finish),
	})

	if err != nil {
		s.errorf(err, "Error creating session entry")
	}
}

func (s *Session) UpdateProposal(d ProposalData) {
	s.updateSession(func(session *data.Session) {
		session.Indexes = d.Indexes
		session.Root = sql.NullString{
			String: d.Root,
			Valid:  d.Root != "",
		}
	})
}

func (s *Session) UpdateAcceptance(d AcceptanceData) {
	s.updateSession(func(session *data.Session) {
		session.Accepted = make([]string, 0, len(d.Acceptances))
		for p := range d.Acceptances {
			session.Accepted = append(session.Accepted, p)
		}
	})
}

func (s *Session) UpdateSignature(d SignatureData) {
	s.updateSession(func(session *data.Session) {
		session.Signature = sql.NullString{
			String: d.Signature,
			Valid:  true,
		}
	})
}

func (s *Session) Success() {
	s.updateSession(func(session *data.Session) {
		session.Status = int(types.Status_Success)
	})
}

func (s *Session) Failed() {
	s.updateSession(func(session *data.Session) {
		session.Status = int(types.Status_Failed)
	})
}

func (s *Session) updateSession(f func(session *data.Session)) {
	if !s.initialized {
		return
	}

	session, err := s.storage.SessionQ().SessionByID(int64(s.id), true)
	if err != nil {
		s.errorf(err, "Error selecting session entry")
		return
	}

	f(session)

	if err := s.storage.SessionQ().Update(session); err != nil {
		s.errorf(err, "Error updating session entry")
	}
}

func (s *Session) infof(msg string, args ...interface{}) {
	s.Infof("[Session %d] - %s", s.id, fmt.Sprintf(msg, args))
}

func (s *Session) errorf(err error, msg string, args ...interface{}) {
	s.WithError(err).Errorf("[Session %d] - %s", s.id, fmt.Sprintf(msg, args))
}
