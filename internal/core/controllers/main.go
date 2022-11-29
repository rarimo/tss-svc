package controllers

import (
	"context"
	goerr "errors"

	"github.com/bnb-chain/tss-lib/tss"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrSenderIsNotProposer  = goerr.New("party is not proposer")
	ErrUnsupportedContent   = goerr.New("unsupported content")
	ErrInvalidRequestType   = goerr.New("invalid request type")
	ErrSenderHasNotAccepted = goerr.New("sender has not accepted proposal")
)

type (
	IController interface {
		Receive(request *types.MsgSubmitRequest) error
		Run(ctx context.Context)
		WaitFor()
		Next() IController
		Type() types.ControllerType
	}

	LocalSessionData struct {
		SessionId          uint64
		Processing         bool
		SessionType        types.SessionType
		Proposer           rarimo.Party
		Old                *core.InputSet
		New                *core.InputSet
		Indexes            []string
		Root               string
		Acceptances        map[string]struct{}
		AcceptedPartyIds   tss.SortedPartyIDs
		NewGlobalPublicKey string
		OperationSignature string
		KeySignature       string
	}
)
