package step

import (
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

const (
	ProposingIndex = 0
	AcceptingIndex = 1
	SigningIndex   = 2
)

// Step stores information about current step and its start/finish
// and provides moving to the next step if available.
type Step struct {
	params *local.Params

	stepType   types.StepType
	startBlock uint64
	endBlock   uint64
}

func NewLastStep(end uint64) *Step {
	return &Step{
		params:     nil,
		stepType:   types.StepType_Signing,
		startBlock: 0,
		endBlock:   end,
	}
}

func NewStep(params *local.Params, startBlock uint64) *Step {
	return &Step{
		params:     params,
		stepType:   types.StepType_Proposing,
		startBlock: startBlock,
		endBlock:   startBlock + params.Step(ProposingIndex).Duration - 1,
	}
}

func (s *Step) IsStarted(height uint64) bool {
	return s.startBlock <= height
}

func (s *Step) Type() types.StepType {
	return s.stepType
}

func (s *Step) Next(height uint64) bool {
	if height > s.endBlock {
		switch s.stepType {
		case types.StepType_Signing:
			return false
		case types.StepType_Proposing:
			s.stepType = types.StepType_Accepting
			s.startBlock = s.endBlock + 1
			s.endBlock = s.startBlock + s.params.Step(AcceptingIndex).Duration - 1
			return true
		case types.StepType_Accepting:
			s.stepType = types.StepType_Signing
			s.startBlock = s.endBlock + 1
			s.endBlock = s.startBlock + s.params.Step(SigningIndex).Duration - 1
			return true
		}
	}

	return false
}

func (s *Step) EndAllBlock() uint64 {
	switch s.stepType {
	case types.StepType_Proposing:
		return s.startBlock + s.params.Step(ProposingIndex).Duration + s.params.Step(AcceptingIndex).Duration + s.params.Step(SigningIndex).Duration - 1
	case types.StepType_Accepting:
		return s.startBlock + s.params.Step(AcceptingIndex).Duration + s.params.Step(SigningIndex).Duration - 1
	case types.StepType_Signing:
		return s.startBlock + s.params.Step(SigningIndex).Duration - 1
	}
	return 0
}
