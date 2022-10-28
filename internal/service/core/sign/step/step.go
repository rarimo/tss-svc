package step

import (
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/core/sign"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
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
		endBlock:   startBlock + params.Step(sign.StepProposingIndex).Duration,
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
			s.endBlock = s.startBlock + s.params.Step(sign.StepAcceptingIndex).Duration
		case types.StepType_Accepting:
			s.stepType = types.StepType_Signing
			s.startBlock = s.endBlock + 1
			s.endBlock = s.startBlock + s.params.Step(sign.StepSigningIndex).Duration
		}
	}

	return false
}
