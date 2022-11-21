package core

import (
	"context"

	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/timer"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

const (
	ProposingIndex = 0
	AcceptingIndex = 1
	SigningIndex   = 2
	FinishingIndex = 3
	ReshareIndex   = 4
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
	SessionID() uint64
	StepType() types.StepType
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

type (
	ProposalData struct {
		Indexes []string
		Root    string
		Reshare bool
	}

	AcceptanceData struct {
		Indexes     []string
		Root        string
		Acceptances map[string]struct{}
		Reshare     bool
	}

	SignatureData struct {
		Indexes   []string
		Root      string
		Signature string
		Reshare   bool
	}
)

func Run(cfg config.Config) {
	factory := NewControllerFactory(cfg)
	timer := timer.NewTimer(cfg)
	fBounds := NewBounds(cfg.Session().StartBlock, local.NewParams(cfg).Step(FinishingIndex).Duration)
	if fBounds.start < timer.CurrentBlock() {
		panic("invalid start block")
	}

	controller := factory.GetEmptyController(factory.GetFinishController(SignatureData{}, fBounds), NewBoundsWithEnd(timer.CurrentBlock(), fBounds.start-1))
	timer.SubscribeToBlocks("manager", NewManager(controller).NewBlock)
}

func RunKeygen(cfg config.Config) {
	factory := NewControllerFactory(cfg)
	timer := timer.NewTimer(cfg)
	controller := factory.GetKeygenController(NewBoundsWithEnd(timer.CurrentBlock(), cfg.Session().StartBlock-1))
	timer.SubscribeToBlocks("manager", NewManager(controller).NewBlock)
}
