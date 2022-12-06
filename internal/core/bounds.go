package core

import (
	"sync"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

const (
	SessionDuration    = 35
	ProposalDuration   = 5
	AcceptanceDuration = 5
	SignDuration       = 5
	ReshareDuration    = 5
	KeygenDuration     = 5
)

// Default: 0-5 proposal 6-11 acceptance 12-17 sign 17-35 finish
// Reshare 0-5 proposal 6-11 acceptance 12-17 reshare 18-23 sign 24-29 sign 30-35 finish

var durationByControllers = map[types.ControllerType]uint64{
	types.ControllerType_CONTROLLER_KEYGEN:     KeygenDuration,
	types.ControllerType_CONTROLLER_PROPOSAL:   ProposalDuration,
	types.ControllerType_CONTROLLER_ACCEPTANCE: AcceptanceDuration,
	types.ControllerType_CONTROLLER_SIGN:       SignDuration,
	types.ControllerType_CONTROLLER_RESHARE:    ReshareDuration,
}

type Bounds struct {
	Start uint64
	End   uint64
}

// BoundsManager is responsible for managing controllers bounds
type BoundsManager struct {
	mu           sync.Mutex
	SessionStart uint64
	SessionEnd   uint64
	bounds       []*Bounds
}

func NewBoundsManager(start uint64) *BoundsManager {
	return &BoundsManager{
		SessionStart: start,
		SessionEnd:   start + SessionDuration,
		bounds:       make([]*Bounds, 0, 6),
	}
}

func (b *BoundsManager) NextController(t types.ControllerType) *Bounds {
	start := b.SessionStart
	if len(b.bounds) > 0 {
		start = b.bounds[len(b.bounds)-1].End + 1
	}

	bound := &Bounds{
		Start: start,
		End:   b.SessionEnd,
	}

	if t != types.ControllerType_CONTROLLER_FINISH {
		bound.End = start + durationByControllers[t]
	}

	b.bounds = append(b.bounds, bound)
	return bound
}

func (b *BoundsManager) Current() *Bounds {
	if len(b.bounds) > 0 {
		return b.bounds[len(b.bounds)-1]
	}

	return &Bounds{
		Start: b.SessionStart,
		End:   b.SessionEnd,
	}
}
