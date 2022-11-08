package session

import (
	"database/sql"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/data"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// Session controls information about current session and saves all in db.
type Session struct {
	id uint64

	status     types.Status
	startBlock uint64
	endBlock   uint64
	proposer   rarimo.Party

	root     string
	indexes  []string
	accepted []string
	sign     string

	proposal   chan *Proposal
	acceptance chan *Acceptance
	signature  chan *Signature

	storage *pg.Storage
}

func NewSession(
	id uint64,
	startBlock uint64,
	endBlock uint64,
	proposer rarimo.Party,
	storage *pg.Storage,
) ISession {
	s := &Session{
		id:         id,
		status:     types.Status_Processing,
		startBlock: startBlock,
		endBlock:   endBlock,
		proposer:   proposer,
		indexes:    []string{},
		proposal:   make(chan *Proposal, 1),
		acceptance: make(chan *Acceptance, 1),
		signature:  make(chan *Signature, 1),
		storage:    storage,
	}

	err := storage.SessionQ().Upsert(&data.Session{
		ID:     int64(s.id),
		Status: int(s.status),
		Proposer: sql.NullString{
			String: s.proposer.PubKey,
			Valid:  true,
		},
		BeginBlock: int64(s.startBlock),
		EndBlock:   int64(s.endBlock),
	})

	if err != nil {
		panic(err)
	}

	return s
}

func (s *Session) ID() uint64 {
	return s.id
}

func (s *Session) Root() string {
	return s.root
}

func (s *Session) Acceptances() []string {
	return s.accepted
}

func (s *Session) Indexes() []string {
	return s.indexes
}

func (s *Session) Signature() string {
	return s.sign
}

func (s *Session) Proposer() rarimo.Party {
	return s.proposer
}

func (s *Session) Start() uint64 {
	return s.startBlock
}

func (s *Session) End() uint64 {
	return s.endBlock
}

func (s *Session) GetProposalChanel() chan *Proposal {
	return s.proposal
}

func (s *Session) GetAcceptanceChanel() chan *Acceptance {
	return s.acceptance
}

func (s *Session) GetSignatureChanel() chan *Signature {
	return s.signature
}

func (s *Session) IsStarted(height uint64) bool {
	return s.startBlock <= height
}

func (s *Session) IsFinished(height uint64) bool {
	return s.endBlock < height
}

func (s *Session) IsFailed() bool {
	return s.status == types.Status_Failed
}

func (s *Session) IsSuccess() bool {
	return s.status == types.Status_Success
}

func (s *Session) IsProcessing() bool {
	return s.status == types.Status_Processing
}

// FinishProposal tires to finish proposal step if there is any information received.
// Responds with true if successful
func (s *Session) FinishProposal() bool {
	if s.status != types.Status_Processing {
		return s.status == types.Status_Success
	}

	select {
	case info, ok := <-s.proposal:
		if !ok {
			return false
		}

		if len(info.Indexes) == 0 {
			s.status = types.Status_Success
		}

		s.root = info.Root
		s.indexes = info.Indexes
		err := s.updateEntry(func(entry *data.Session) {
			entry.Status = int(s.status)
			entry.Indexes = info.Indexes
			entry.Root = sql.NullString{
				String: info.Root,
				Valid:  true,
			}
		})

		if err != nil {
			panic(err)
		}
	default:
		return false
	}
	return true
}

// FinishAcceptance tires to finish acceptance step if there is any information received.
// Responds with true if successful
func (s *Session) FinishAcceptance() bool {
	if s.status != types.Status_Processing {
		return s.status == types.Status_Success
	}

	select {
	case info, ok := <-s.acceptance:
		if !ok {
			return false
		}

		s.accepted = info.Accepted
		err := s.updateEntry(func(entry *data.Session) {
			entry.Accepted = info.Accepted
		})

		if err != nil {
			panic(err)
		}
	default:
		return false
	}

	return true
}

// FinishSign tires to finish signing step if there is any information received.
// Responds with true if successful.
// After successful finishing of signing step session will be marked as success
func (s *Session) FinishSign() bool {
	if s.status != types.Status_Processing {
		return s.status == types.Status_Success
	}

	select {
	case info, ok := <-s.signature:
		if !ok {
			return false
		}

		s.status = types.Status_Success
		s.sign = info.Signature
		err := s.updateEntry(func(entry *data.Session) {
			entry.Signed = info.Signed
			entry.Signature = sql.NullString{
				String: info.Signature,
				Valid:  true,
			}
			entry.Status = int(s.status)
		})

		if err != nil {
			panic(err)
		}
	default:
		return false
	}

	return true
}

// Fail updates session status to types.Status_Failed
func (s *Session) Fail() {
	if s.status != types.Status_Processing && s.status != types.Status_Pending {
		return
	}

	s.status = types.Status_Failed
	err := s.updateEntry(func(entry *data.Session) {
		entry.Status = int(s.status)
	})

	if err != nil {
		panic(err)
	}
}

func (s *Session) updateEntry(updateF func(entry *data.Session)) error {
	entry, err := s.storage.SessionQ().SessionByID(int64(s.id), false)
	if err != nil {
		return err
	}

	updateF(entry)

	return s.storage.SessionQ().Update(entry)
}
