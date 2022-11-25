package controllers

import (
	"context"
	goerr "errors"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

const (
	ProposingIndex = 0
	AcceptingIndex = 1
	SigningIndex   = 2
	FinishingIndex = 3
	ReshareIndex   = 4
)

var (
	ErrSenderIsNotProposer  = goerr.New("party is not proposer")
	ErrUnsupportedContent   = goerr.New("unsupported content")
	ErrInvalidRequestType   = goerr.New("invalid request type")
	ErrSenderHasNotAccepted = goerr.New("sender has not accepted proposal")
)

type IController interface {
	Receive(request *types.MsgSubmitRequest) error
	Run(ctx context.Context)
	WaitFor()
	Next() IController
	Bounds() *core.Bounds
}

type (
	LocalSessionData struct {
		Proposer  rarimo.Party
		SessionId uint64
	}

	LocalProposalData struct {
		LocalSessionData
		NewSet types.Set
		OldSet types.Set
	}

	LocalAcceptanceData struct {
		LocalProposalData
		Accepted map[string]struct{}
	}

	LocalSignatureData struct {
		LocalAcceptanceData
		Signature string
	}
)
