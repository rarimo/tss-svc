package handlers

// BlockHandler listens to the new blocks received from chanel and upgrading the timer state
type BlockHandler struct {
	op <-chan uint64
}

func NewBlockHandler(op <-chan uint64) *BlockHandler {
	s := &BlockHandler{
		op: op,
	}

	return s
}

func (o *BlockHandler) listen() {

}
