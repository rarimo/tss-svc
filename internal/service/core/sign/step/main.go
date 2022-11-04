package step

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/internal/service/core"
)

type IController interface {
	core.IReceive
	Run(ctx context.Context)
	WaitFinish()
}
