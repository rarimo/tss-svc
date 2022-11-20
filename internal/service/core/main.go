package core

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/auth"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/timer"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

const (
	ProposingIndex = 0
	AcceptingIndex = 1
	SigningIndex   = 2
	FinishingIndex = 2
)

type IGlobalReceiver interface {
	Receive(request *types.MsgSubmitRequest) error
}

type IController interface {
	IGlobalReceiver
	Run(ctx context.Context)
	WaitFor()
	Next() IController
	Start() uint64
	End() uint64
}

type bounds struct {
	start  uint64
	finish uint64
}

func (b *bounds) Start() uint64 {
	return b.start
}

func (b *bounds) End() uint64 {
	return b.finish
}

func NewBounds(start, duration uint64) *bounds {
	return &bounds{
		start:  start,
		finish: start + duration,
	}
}

func NewBoundsWithEnd(start, end uint64) *bounds {
	return &bounds{
		start:  start,
		finish: end,
	}
}

type defaultController struct {
	*logan.Entry
	*connectors.ConfirmConnector
	*connectors.BroadcastConnector
	auth   *auth.RequestAuthorizer
	rarimo *grpc.ClientConn
	secret *local.Secret
	params *local.Params
}

func NewDefaultController(cfg config.Config) *defaultController {
	return &defaultController{
		Entry:              cfg.Log(),
		ConfirmConnector:   connectors.NewConfirmConnector(cfg),
		BroadcastConnector: connectors.NewBroadcastConnector(cfg),
		auth:               auth.NewRequestAuthorizer(cfg),
		rarimo:             cfg.Cosmos(),
		secret:             local.NewSecret(cfg),
		params:             local.NewParams(cfg),
	}
}

func Run(cfg config.Config) {
	factory := NewControllerFactory(cfg)
	timer := timer.NewTimer(cfg)
	fBounds := NewBounds(cfg.Session().StartBlock, local.NewParams(cfg).Step(FinishingIndex).Duration)
	if fBounds.start < timer.CurrentBlock() {
		panic("invalid start block")
	}

	controller := factory.GetEmptyController(factory.GetFinishController(1, types.SignatureData{}, fBounds), NewBoundsWithEnd(timer.CurrentBlock(), fBounds.start-1))
	timer.SubscribeToBlocks("manager", NewManager(controller).NewBlock)
}

func RunKeygen(cfg config.Config) {
	factory := NewControllerFactory(cfg)
	timer := timer.NewTimer(cfg)
	controller := factory.GetKeygenController(NewBoundsWithEnd(timer.CurrentBlock(), cfg.Session().StartBlock-1))
	timer.SubscribeToBlocks("manager", NewManager(controller).NewBlock)
}
