package controllers

import (
	"context"
	goerr "errors"

	"github.com/rarimo/tss-svc/pkg/types"
)

var (
	ErrSenderIsNotProposer = goerr.New("party is not proposer")
	ErrUnsupportedContent  = goerr.New("unsupported content")
	ErrInvalidRequestType  = goerr.New("invalid request type")
	ErrSenderIsNotSigner   = goerr.New("sender is no a current signer or has not accepted the proposal")
)

type (
	// IController interface represents the smallest independent part of flow.
	IController interface {
		// Receive accepts all incoming requests
		Receive(ctx context.Context, request *types.MsgSubmitRequest) error
		// Run will execute controller logic in separate goroutine
		Run(ctx context.Context)
		// WaitFor should be used to wait while controller finishes after canceling context.
		WaitFor()
		// Next returns next controller depending on current state or nil if session ended.
		Next() IController
		Type() types.ControllerType
	}
)
