package listeners

import (
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/timer"
)

// BlockListener is listening to the new blocks received from chanel and upgrading the timer state
type BlockListener struct {
	op    <-chan uint64
	timer *timer.Timer
}

func NewBlockListener(op <-chan uint64, t *timer.Timer) *BlockListener {
	s := &BlockListener{
		op:    op,
		timer: t,
	}

	s.listen()
	return s
}

func (o *BlockListener) listen() {
	go func() {
		for {
			height, ok := <-o.op
			if !ok {
				break
			}

			o.timer.NewBlock(height)
		}
	}()
}
