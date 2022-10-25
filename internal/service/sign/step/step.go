package step

import (
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// Step stores information about current step and its start/finish
// and provides moving to the next step if available.
type Step struct {
	params *rarimo.Params

	stepType   types.StepType
	startBlock uint64
	endBlock   uint64
}

func NewStep(params *rarimo.Params, startBlock uint64) *Step {
	return &Step{
		params:     params,
		stepType:   types.StepType_Proposing,
		startBlock: startBlock,
		endBlock:   startBlock + params.Steps[sign.StepProposingIndex].Duration,
	}
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
			s.endBlock = s.startBlock + s.params.Steps[sign.StepAcceptingIndex].Duration
		case types.StepType_Accepting:
			s.stepType = types.StepType_Signing
			s.startBlock = s.endBlock + 1
			s.endBlock = s.startBlock + s.params.Steps[sign.StepSigningIndex].Duration
		}
	}

	return false
}
