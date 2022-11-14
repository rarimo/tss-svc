package pool

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

const (
	OpServiceName    = "op-subscriber"
	OpQueryTransfer  = "tm.event='Tx' AND new_operation.operation_type='TRANSFER'"
	OpQueryChangeKey = "tm.event='Tx' AND new_operation.operation_type='CHANGE_KEY'"
	OpPoolSize       = 1000
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

// NewChangeKeyOperationSubscriber creates the subscriber instance for listening new change key operations
func NewChangeKeyOperationSubscriber(cfg config.Config) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   NewPool(cfg),
		log:    cfg.Log(),
		client: cfg.Tendermint(),
		query:  OpQueryChangeKey,
	}
}

func (o *OperationSubscriber) Run() {
	out, err := o.client.Subscribe(context.Background(), OpServiceName, o.query, OpPoolSize)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			c, ok := <-out
			if !ok {
				if err := o.client.Unsubscribe(context.Background(), OpServiceName, o.query); err != nil {
					o.log.WithError(err).Error("error unsubscribing from new operations")
				}
				break
			}

			for _, index := range c.Events[fmt.Sprintf("%s.%s", rarimo.EventTypeNewOperation, rarimo.AttributeKeyOperationId)] {
				o.log.Debugf("New operation found index=%s", index)
				o.pool.Add(index)
			}

		}
	}()
}
