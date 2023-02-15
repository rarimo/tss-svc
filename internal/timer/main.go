package timer

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
)

// Timer implements the singleton pattern
var timer *Timer

type BlockNotifier func(height uint64) error

// Timer provides the source for timestamping all operations in the tss system.
// Use Notifier to receive notification about new blocks in your service
type Timer struct {
	currentBlock uint64
	toNotify     map[string]BlockNotifier
	log          *logan.Entry
}

// NewTimer returns new Timer but only once because Timer implements the singleton pattern for simple usage as
// the same instance in all injections.
func NewTimer(cfg config.Config) *Timer {
	if timer == nil {
		info, err := cfg.Tendermint().Status(context.TODO())
		if err != nil {
			panic(err)
		}

		timer = &Timer{
			currentBlock: uint64(info.SyncInfo.LatestBlockHeight),
			toNotify:     make(map[string]BlockNotifier),
			log:          cfg.Log(),
		}
	}

	return timer
}

// GetTimer returns existing instance of timer. Be aware to initialize it before using that function.
func GetTimer() *Timer {
	return timer
}

// Only for internal usage in block subscriber
func (t *Timer) newBlock(height uint64) {
	t.currentBlock = height
	go t.notifyAll(height)
}

func (t *Timer) CurrentBlock() uint64 {
	return t.currentBlock
}

// SubscribeToBlocks adds receiver method to notify fot the new block events
func (t *Timer) SubscribeToBlocks(name string, f BlockNotifier) {
	t.toNotify[name] = f
	go t.notify(t.currentBlock, name, f)
}

func (t *Timer) notifyAll(height uint64) {
	for name, f := range t.toNotify {
		t.notify(height, name, f)
	}
}

func (t *Timer) notify(height uint64, name string, f BlockNotifier) {
	if err := f(height); err != nil {
		t.log.WithError(err).Errorf("[Block] Got an error notifying for the new block %s", name)
	}
}
