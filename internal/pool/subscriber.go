package pool

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
)

const (
	OpServiceName   = "op-subscriber"
	OpQueryTransfer = "tm.event='Tx' AND operation_approved.operation_type='TRANSFER'"
	OpPoolSize      = 1000
)

// OperationSubscriber subscribes to the NewOperation events on the tendermint core.
type OperationSubscriber struct {
	pool   *Pool
	client *http.HTTP
	query  string
	log    *logan.Entry
}

// NewTransferOperationSubscriber creates the subscriber instance for listening new transfer operations
func NewTransferOperationSubscriber(cfg config.Config) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   NewPool(cfg),
		log:    cfg.Log(),
		client: cfg.Tendermint(),
		query:  OpQueryTransfer,
	}
}

func (o *OperationSubscriber) Run() {
	go func() {
		for {
			o.runner()
			o.log.Info("[Pool] Resubscribing to the pool...")
		}
	}()
}

func (o *OperationSubscriber) runner() {
	out, err := o.client.Subscribe(context.Background(), OpServiceName, o.query, OpPoolSize)
	if err != nil {
		panic(err)
	}

	for {
		c, ok := <-out
		if !ok {
			if err := o.client.Unsubscribe(context.Background(), OpServiceName, o.query); err != nil {
				o.log.WithError(err).Error("[Pool] Failed to unsubscribe from new operations")
			}
			break
		}

		for _, index := range c.Events[fmt.Sprintf("%s.%s", rarimo.EventTypeNewOperation, rarimo.AttributeKeyOperationId)] {
			o.log.Infof("[Pool] New operation found index=%s", index)
			if err := o.pool.Add(index); err != nil {
				o.log.WithError(err).Error("error adding operation to the pool")
			}

		}
	}
}
