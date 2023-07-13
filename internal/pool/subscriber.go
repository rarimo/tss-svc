package pool

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
)

const (
	OpServiceName                  = "op-subscriber"
	OpQueryTransfer                = "tm.event='Tx' AND operation_approved.operation_type='TRANSFER'"
	OpQueryFeeManagement           = "tm.event='NewBlock' AND operation_approved.operation_type='FEE_TOKEN_MANAGEMENT'"
	OpQueryContractUpgrade         = "tm.event='NewBlock' AND operation_approved.operation_type='CONTRACT_UPGRADE'"
	OpQueryIdentityDefaultTransfer = "tm.event='Tx' AND operation_approved.operation_type='IDENTITY_DEFAULT_TRANSFER'"
	OpPoolSize                     = 1000
)

// OperationSubscriber subscribes to the NewOperation events on the tendermint core.
type OperationSubscriber struct {
	pool   *Pool
	client *http.HTTP
	query  string
	log    *logan.Entry
}

// NewIdentityTransferOperationSubscriber creates the subscriber instance for listening new identity transfer operations
func NewIdentityTransferOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryIdentityDefaultTransfer,
	}
}

// NewContractUpgradeOperationSubscriber creates the subscriber instance for listening new contract upgrades operations
func NewContractUpgradeOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryContractUpgrade,
	}
}

// NewFeeManagementOperationSubscriber creates the subscriber instance for listening new fee token management operations
func NewFeeManagementOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryFeeManagement,
	}
}

// NewTransferOperationSubscriber creates the subscriber instance for listening new transfer operations
func NewTransferOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryTransfer,
	}
}

func (o *OperationSubscriber) Run() {
	go func() {
		for {
			o.log.Infof("[Pool] Subscribing to the pool. Query: %s", o.query)
			o.runner()
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
			o.log.Info("[Pool] WS unsubscribed. Resubscribing...")
			if err := o.client.Unsubscribe(context.Background(), OpServiceName, o.query); err != nil {
				o.log.WithError(err).Error("[Pool] Failed to unsubscribe from new operations")
			}
			break
		}

		for _, index := range c.Events[fmt.Sprintf("%s.%s", rarimo.EventTypeOperationApproved, rarimo.AttributeKeyOperationId)] {
			o.log.Infof("[Pool] New operation found index=%s", index)
			if err := o.pool.Add(index); err != nil {
				o.log.WithError(err).Error("error adding operation to the pool")
			}
		}
	}
}
