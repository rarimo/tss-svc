package core

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type EmptyController struct {
	*bounds
	next IController
}

func NewEmptyController(next IController, bounds *bounds) IController {
	return &EmptyController{
		next:   next,
		bounds: bounds,
	}
}

var _ IController = &EmptyController{}

func (e *EmptyController) Receive(request *types.MsgSubmitRequest) error {
	return nil
}

func (e *EmptyController) Run(ctx context.Context) {}

func (e *EmptyController) WaitFor() {}

func (e *EmptyController) Next() IController {
	return e.next
}
