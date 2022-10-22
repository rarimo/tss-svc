package session

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

type Notifier func(height uint64) error

type Timer struct {
	currentBlock uint64
	toNotify     map[string]Notifier
	log          *logan.Entry
}

func NewTimer(cfg config.Config) *Timer {
	return &Timer{
		toNotify: make(map[string]Notifier),
		log:      cfg.Log(),
	}
}

func (t *Timer) NewBlock(height uint64) {
	t.currentBlock = height
	go t.notify(height)
}

func (t *Timer) notify(height uint64) {
	for name, f := range t.toNotify {
		if err := f(height); err != nil {
			t.log.WithError(err).Errorf("got an error notifying %s", name)
		}
	}
}

func (t *Timer) CurrentBlock() uint64 {
	return t.currentBlock
}

func (t *Timer) SubscribeToNotification(name string, f Notifier) {
	t.toNotify[name] = f
}
