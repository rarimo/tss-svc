package session

import rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"

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

// ISession is responsible for managing session time bounds, storing session intermediate information
// and submits data to the database.
// There are two implementations: DefaultSession and Session. The fist one is used to mock the session where
// the party will not be a part of.
type ISession interface {
	ID() uint64
	Root() string
	Indexes() []string
	Signature() string
	Proposer() rarimo.Party
	Start() uint64
	End() uint64
	GetProposalChanel() chan *Proposal
	GetAcceptanceChanel() chan *Acceptance
	GetSignatureChanel() chan *Signature
	IsStarted(height uint64) bool
	IsFinished(height uint64) bool
	IsFailed() bool
	IsSuccess() bool
	IsProcessing() bool
	FinishProposal() bool
	FinishAcceptance() bool
	FinishSign() bool
	Fail()
}
