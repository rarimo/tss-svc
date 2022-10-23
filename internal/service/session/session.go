package session

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"sync"

	"github.com/anyswap/FastMulThreshold-DSA/crypto"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/data"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/session/params"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/session/timer"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

const (
	StepsAmount        = 3
	StepProposingIndex = 0
	StepAcceptingIndex = 1
	StepSigningIndex   = 2
	ComponentName      = "session"
)

var (
	ErrUnsupportedParameters = errors.New("unsupported parameters")
)

type Session struct {
	mu sync.Mutex
	*types.Session
	startBlock uint64
	endBlock   uint64

	storage *pg.Storage
	log     *logan.Entry

	paramsStorage *params.Storage
	timer         timer.Timer
}

// NextBlock receives next block notifications from timer
func (s *Session) NextBlock(height uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if height >= s.startBlock && s.Status == types.Status_Pending {
		s.Status = types.Status_Processing
	}

	// TODO chane after implementing tss logic
	if height >= s.endBlock {
		s.Status = types.Status_Success
		s.next()
		return nil
	}

	s.tryStepAccepting(height)
	s.tryStepSigning(height)

	return nil
}

func (s *Session) tryStepAccepting(height uint64) {
	if height >= s.Steps[StepAcceptingIndex].StartBlock {
		s.CurrentStep = types.StepType_Accepting
	}
}

func (s *Session) tryStepSigning(height uint64) {
	if height >= s.Steps[StepSigningIndex].StartBlock {
		s.CurrentStep = types.StepType_Signing
	}
}

// Next moves to the next session
func (s *Session) next() {
	params := s.paramsStorage.GetParams()
	if len(params.Steps) != StepsAmount {
		panic(ErrUnsupportedParameters)
	}

	steps := []*types.Step{
		getStepProposing(s.endBlock, params.Steps),
		getStepAccepting(s.endBlock, params.Steps),
		getStepSigning(s.endBlock, params.Steps),
	}

	proposer := s.nextProposer(params)

	next := &types.Session{
		Id:          s.Id + 1,
		Status:      types.Status_Pending,
		Steps:       steps,
		CurrentStep: types.StepType_Proposing,
		Pool: &types.Pool{
			Proposer: proposer.PubKey,
		},
	}

	err := s.storage.SessionQ().Insert(&data.Session{
		ID:     int64(s.Id),
		Status: int(s.Status),
		Proposer: sql.NullString{
			String: proposer.PubKey,
			Valid:  true,
		},
		BeginBlock: int64(steps[StepProposingIndex].StartBlock),
		EndBlock:   int64(steps[StepSigningIndex].EndBlock),
	})

	if err != nil {
		panic(err)
	}

	s.Session = next
	s.startBlock = steps[StepProposingIndex].StartBlock
	s.endBlock = steps[StepSigningIndex].EndBlock
	s.timer.SubscribeToBlocks(ComponentName, s.NextBlock)
}

func (s *Session) nextProposer(params *rarimo.Params) *rarimo.Party {
	sigBytes := hexutil.MustDecode(s.Signature)
	stepBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(stepBytes, s.Id)
	hash := crypto.Keccak256(sigBytes, stepBytes)
	return params.Parties[int(hash[len(hash)-1])%len(params.Parties)]
}

func getStepProposing(lastSessionEnd uint64, durations []*rarimo.Step) *types.Step {
	if uint32(durations[0].Type) != uint32(types.StepType_Proposing) {
		panic(ErrUnsupportedParameters)
	}

	return &types.Step{
		StartBlock: lastSessionEnd + 1,
		EndBlock:   lastSessionEnd + 1 + durations[StepProposingIndex].Duration,
		Type:       types.StepType_Proposing,
	}
}

func getStepAccepting(lastSessionEnd uint64, durations []*rarimo.Step) *types.Step {
	if uint32(durations[StepAcceptingIndex].Type) != uint32(types.StepType_Accepting) {
		panic(ErrUnsupportedParameters)
	}

	return &types.Step{
		StartBlock: lastSessionEnd + 1 + durations[StepProposingIndex].Duration + 1,
		EndBlock:   lastSessionEnd + 1 + durations[StepProposingIndex].Duration + 1 + durations[StepAcceptingIndex].Duration,
		Type:       types.StepType_Accepting,
	}
}

func getStepSigning(lastSessionEnd uint64, durations []*rarimo.Step) *types.Step {
	if uint32(durations[StepSigningIndex].Type) != uint32(types.StepType_Signing) {
		panic(ErrUnsupportedParameters)
	}

	return &types.Step{
		StartBlock: lastSessionEnd + 1 + durations[StepProposingIndex].Duration + 1 + durations[StepAcceptingIndex].Duration + 1,
		EndBlock:   lastSessionEnd + 1 + durations[StepProposingIndex].Duration + 1 + durations[StepAcceptingIndex].Duration + 1 + durations[StepSigningIndex].Duration,
		Type:       types.StepType_Signing,
	}
}
