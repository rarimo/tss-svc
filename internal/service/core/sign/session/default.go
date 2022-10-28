package session

import (
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
)

// DefaultSession stores id and end block of the session and used to schedule the real session start
// during launch of the main service.
type DefaultSession struct {
	id  uint64
	end uint64
}

func NewDefaultSession(id uint64, end uint64) ISession {
	return &DefaultSession{
		id:  id,
		end: end,
	}
}

func (p *DefaultSession) ID() uint64 {
	return p.id
}

func (p *DefaultSession) Root() string {
	return ""
}

func (p *DefaultSession) Indexes() []string {
	return []string{}
}

func (p *DefaultSession) Signature() string {
	return ""
}

func (p *DefaultSession) Proposer() rarimo.Party {
	return rarimo.Party{}
}

func (p *DefaultSession) Start() uint64 {
	return 0
}

func (p *DefaultSession) End() uint64 {
	return p.end
}

func (p *DefaultSession) GetProposalChanel() chan *Proposal {
	return nil
}

func (p *DefaultSession) GetAcceptanceChanel() chan *Acceptance {
	return nil
}

func (p *DefaultSession) GetSignatureChanel() chan *Signature {
	return nil
}

func (p *DefaultSession) IsStarted(height uint64) bool {
	return true
}

func (p *DefaultSession) IsFinished(height uint64) bool {
	return height > p.end
}

func (p *DefaultSession) IsFailed() bool {
	return false
}

func (p *DefaultSession) IsSuccess() bool {
	return true
}

func (p *DefaultSession) IsProcessing() bool {
	return false
}

func (p *DefaultSession) FinishProposal() bool {
	return true
}

func (p *DefaultSession) FinishAcceptance() bool {
	return true
}

func (p *DefaultSession) FinishSign() bool {
	return true
}

func (p *DefaultSession) Fail() {}
