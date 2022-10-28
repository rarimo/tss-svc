package pool

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

// TODO also listening other operations
const (
	OpServiceName = "op-subscriber"
	OpQuery       = "tm.event='Tx' AND new_operation.operation_type='TRANSFER,CHANGE_KEY'"
	OpPoolSize    = 1000
)

// OperationSubscriber subscribes to the NewOperation events on the tendermint core.
type OperationSubscriber struct {
	pool   *Pool
	client *http.HTTP
	log    *logan.Entry
}

func NewOperationSubscriber(cfg config.Config) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   NewPool(cfg),
		log:    cfg.Log(),
		client: cfg.Tendermint(),
	}
}

func (o *OperationSubscriber) Run() {
	out, err := o.client.Subscribe(context.Background(), OpServiceName, OpQuery, OpPoolSize)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			c, ok := <-out
			if !ok {
				if err := o.client.Unsubscribe(context.Background(), OpServiceName, OpQuery); err != nil {
					o.log.WithError(err).Error("error unsubscribing from new operations")
				}
				break
			}

			for _, index := range c.Events[fmt.Sprintf("%s.%s", rarimo.EventTypeNewOperation, rarimo.AttributeKeyOperationId)] {
				o.log.Infof("New operation found index=%s", index)
				o.pool.Add(index)
			}

		}
	}()
}
