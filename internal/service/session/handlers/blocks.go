package handlers

import "gitlab.com/rarify-protocol/tss-svc/internal/service/session"

// BlockHandler is listening to the new blocks received from chanel and upgrading the timer state
type BlockHandler struct {
	op    <-chan uint64
	timer *session.Timer
}

func NewBlockHandler(op <-chan uint64, t *session.Timer) *BlockHandler {
	s := &BlockHandler{
		op:    op,
		timer: t,
	}

	s.listen()

	return s
}

func (o *BlockHandler) listen() {
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
