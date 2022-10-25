package listeners

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/sign/pool"
)

// OperationListener is listening to the new operations received from chanel and moving them to the pool
type OperationListener struct {
	op   <-chan string
	pool *pool.Pool
	log  *logan.Entry
}

func NewOperationListener(op <-chan string, p *pool.Pool, cfg config.Config) *OperationListener {
	s := &OperationListener{
		op:   op,
		pool: p,
		log:  cfg.Log(),
	}

	s.listen()
	return s
}

func (o *OperationListener) listen() {
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
