package handlers

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/session"
)

// OperationHandler listens to the new operations received from chanel and moving them to the pool
type OperationHandler struct {
	op   <-chan string
	pool *session.Pool
	log  *logan.Entry
}

func NewOperationHandler(op <-chan string, p *session.Pool, cfg config.Config) *OperationHandler {
	s := &OperationHandler{
		op:   op,
		pool: p,
		log:  cfg.Log(),
	}

	s.listen()
	return s
}

func (o *OperationHandler) listen() {
	go func() {
		for {
			id, ok := <-o.op
			if !ok {
				break
			}

			if err := o.pool.Add(id); err != nil {
				o.log.WithError(err).Error("error adding to the pool")
			}
		}
	}()
}
