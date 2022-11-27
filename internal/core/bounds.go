package core

import "gitlab.com/rarify-protocol/tss-svc/pkg/types"

type Bounds struct {
	Start uint64
	End   uint64
}

type BoundsManager struct {
	SessionStart uint64
	SessionEnd   uint64
}

func (*BoundsManager) GetBounds(t types.ControllerType) *Bounds {
	return &Bounds{}
}
