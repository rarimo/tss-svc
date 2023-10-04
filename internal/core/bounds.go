package core

import (
	"sync"

	"github.com/rarimo/tss-svc/pkg/types"
)

// Default: 0-2 proposal 3-5 acceptance 6-12 sign 13-15 finish
// Keygen: 0-10 11-13 finish
// Reshare 0-2 proposal 3-5 acceptance 6-30 keygen 31-37 sign 38-44 sign 45-47 finish
const (
	DefaultSessionDuration           = 15
	DefaultSessionProposalDuration   = 2
	DefaultSessionAcceptanceDuration = 2
	DefaultSessionSignDuration       = 6

	KeygenSessionDuration       = 13
	KeygenSessionKeygenDuration = 10

	ReshareSessionDuration           = 47
	ReshareSessionProposalDuration   = 2
	ReshareSessionAcceptanceDuration = 2
	ReshareSessionKeygenDuration     = 24
	ReshareSessionSignDuration       = 6
)

type Bounds struct {
	Start uint64
	End   uint64
}

// BoundsManager is responsible for managing controllers bounds
type BoundsManager struct {
	mu                   sync.Mutex
	SessionStart         uint64
	SessionEnd           uint64
	SessionDuration      uint64
	durationByController map[types.ControllerType]uint64
	bounds               []*Bounds
}

func NewBoundsManager(start uint64, sessionType types.SessionType) *BoundsManager {
	switch sessionType {
	case types.SessionType_DefaultSession:
		return &BoundsManager{
			SessionStart:    start,
			SessionDuration: DefaultSessionDuration,
			SessionEnd:      start + DefaultSessionDuration,
			bounds:          make([]*Bounds, 0, 4),
			durationByController: map[types.ControllerType]uint64{
				types.ControllerType_CONTROLLER_PROPOSAL:   DefaultSessionProposalDuration,
				types.ControllerType_CONTROLLER_ACCEPTANCE: DefaultSessionAcceptanceDuration,
				types.ControllerType_CONTROLLER_SIGN:       DefaultSessionSignDuration,
			},
		}
	case types.SessionType_KeygenSession:
		return &BoundsManager{
			SessionStart:    start,
			SessionDuration: KeygenSessionDuration,
			SessionEnd:      start + KeygenSessionDuration,
			bounds:          make([]*Bounds, 0, 2),
			durationByController: map[types.ControllerType]uint64{
				types.ControllerType_CONTROLLER_KEYGEN: KeygenSessionKeygenDuration,
			},
		}
	case types.SessionType_ReshareSession:
		return &BoundsManager{
			SessionStart:    start,
			SessionDuration: ReshareSessionDuration,
			SessionEnd:      start + ReshareSessionDuration,
			bounds:          make([]*Bounds, 0, 6),
			durationByController: map[types.ControllerType]uint64{
				types.ControllerType_CONTROLLER_PROPOSAL:   ReshareSessionProposalDuration,
				types.ControllerType_CONTROLLER_ACCEPTANCE: ReshareSessionAcceptanceDuration,
				types.ControllerType_CONTROLLER_KEYGEN:     ReshareSessionKeygenDuration,
				types.ControllerType_CONTROLLER_SIGN:       ReshareSessionSignDuration,
			},
		}
	}

	// Should not appear
	panic("Invalid session type")
}

func (b *BoundsManager) NextController(t types.ControllerType) *Bounds {
	start := b.SessionStart
	if len(b.bounds) > 0 {
		start = b.bounds[len(b.bounds)-1].End + 1
	}

	bound := &Bounds{
		Start: start,
		End:   b.SessionStart + b.SessionDuration,
	}

	if t != types.ControllerType_CONTROLLER_FINISH {
		bound.End = start + b.durationByController[t]
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
		End:   b.SessionStart + b.SessionDuration,
	}
}
