package session

import (
	"database/sql"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/data"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type (
	Proposal struct {
		Indexes []string
		Root    string
	}

	Acceptance struct {
		Accepted []string
	}

	Signature struct {
		Signed    []string
		Signature string
	}
)

// Session controls information about current session
type Session struct {
	// Default session information
	id uint64

	status     types.Status
	startBlock uint64
	endBlock   uint64
	proposer   *rarimo.Party

	root string

	proposal   chan *Proposal
	acceptance chan *Acceptance
	signature  chan *Signature

	storage *pg.Storage
}

func NewSession(
	id uint64,
	startBlock uint64,
	params *rarimo.Params,
	proposer *rarimo.Party,
	storage *pg.Storage,
) *Session {
	s := &Session{
		id:         id,
		status:     types.Status_Processing,
		startBlock: startBlock,
		endBlock:   startBlock + params.Steps[sign.StepProposingIndex].Duration + 1 + params.Steps[sign.StepAcceptingIndex].Duration + 1 + params.Steps[sign.StepSigningIndex].Duration,
		proposer:   proposer,
		proposal:   make(chan *Proposal, 1),
		acceptance: make(chan *Acceptance, 1),
		signature:  make(chan *Signature, 1),
		storage:    storage,
	}

	err := storage.SessionQ().Insert(&data.Session{
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
		// TODO
		panic(err)
	}

	return s
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

func (s *Session) FinishProposal() bool {
	if s.status != types.Status_Processing {
		return false
	}

	select {
	case info, ok := <-s.proposal:
		if !ok {
			return false
		}

		s.root = info.Root
		err := s.updateEntry(func(entry *data.Session) {
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

func (s *Session) FinishAcceptance() bool {
	if s.status != types.Status_Processing {
		return false
	}

	select {
	case info, ok := <-s.acceptance:
		if !ok {
			return false
		}

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

func (s *Session) FinishSigning() bool {
	if s.status != types.Status_Processing {
		return true
	}

	select {
	case info, ok := <-s.signature:
		if !ok {
			return false
		}

		s.status = types.Status_Success

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
