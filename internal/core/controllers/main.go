package controllers

import (
	"context"
	goerr "errors"

	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

var (
	ErrSenderIsNotProposer          = goerr.New("party is not proposer")
	ErrUnsupportedContent           = goerr.New("unsupported content")
	ErrInvalidRequestType           = goerr.New("invalid request type")
	ErrSenderHasNotAccepted         = goerr.New("sender has not accepted proposal")
	ErrSecretDataAlreadyInitialized = goerr.New("secret data already initialized")
)

type (
	// IController interface represents the smallest independent part of flow.
	IController interface {
		// Receive accepts all incoming requests
		Receive(request *types.MsgSubmitRequest) error
		// Run will execute controller logic in separate goroutine
		Run(ctx context.Context)
		// WaitFor should be used to wait while controller finishes after canceling context.
		WaitFor()
		// Next returns next controller depending on current state or nil if session ended.
		Next() IController
		Type() types.ControllerType
	}

	// LocalSessionData represents all necessary data from current session to be shared between controllers.
	LocalSessionData struct {
		SessionId          uint64
		Processing         bool
		SessionType        types.SessionType
		Proposer           rarimo.Party
		Set                *core.InputSet
		Secret             *secret.TssSecret
		NewSecret          *secret.TssSecret
		Indexes            []string
		Root               string
		Acceptances        map[string]struct{}
		OperationSignature string
		KeySignature       string
		NewParties         []*rarimo.Party
	}
)