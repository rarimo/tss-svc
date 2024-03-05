package pool

import (
	"context"
	"fmt"

	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
)

const (
	OpServiceName                     = "op-subscriber"
	OpQueryTransfer                   = "tm.event='Tx' AND operation_approved.operation_type='TRANSFER'"
	OpQueryFeeManagement              = "tm.event='NewBlock' AND operation_approved.operation_type='FEE_TOKEN_MANAGEMENT'"
	OpQueryContractUpgrade            = "tm.event='NewBlock' AND operation_approved.operation_type='CONTRACT_UPGRADE'"
	OpQueryIdentityDefaultTransfer    = "tm.event='Tx' AND operation_approved.operation_type='IDENTITY_DEFAULT_TRANSFER'"
	OpQueryIdentityGISTTransfer       = "tm.event='Tx' AND operation_approved.operation_type='IDENTITY_GIST_TRANSFER'"
	OpQueryIdentityStateTransfer      = "tm.event='Tx' AND operation_approved.operation_type='IDENTITY_STATE_TRANSFER'"
	OpQueryWorldCoinIdentityTransfer  = "tm.event='Tx' AND operation_approved.operation_type='WORLDCOIN_IDENTITY_TRANSFER'"
	OpQueryIdentityAggregatedTransfer = "tm.event='NewBlock' AND operation_approved.operation_type='IDENTITY_AGGREGATED_TRANSFER'"
	OpPoolSize                        = 1000
)

// OperationSubscriber subscribes to the NewOperation events on the tendermint core.
type OperationSubscriber struct {
	pool   *Pool
	client *http.HTTP
	query  string
	log    *logan.Entry
}

// NewWorldCoinIdentityTransferOperationSubscriber creates the subscriber instance for listening new wordlcoin identity transfer operations
func NewWorldCoinIdentityTransferOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryWorldCoinIdentityTransfer,
	}
}

// NewIdentityAggregatedTransferOperationSubscriber creates the subscriber instance for listening new identity aggregated transfer operations
func NewIdentityAggregatedTransferOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryIdentityAggregatedTransfer,
	}
}

// NewIdentityGISTTransferOperationSubscriber creates the subscriber instance for listening new identity GIST transfer operations
func NewIdentityGISTTransferOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryIdentityGISTTransfer,
	}
}

// NewIdentityStateTransferOperationSubscriber creates the subscriber instance for listening new identity state transfer operations
func NewIdentityStateTransferOperationSubscriber(pool *Pool, tendermint *http.HTTP, log *logan.Entry) *OperationSubscriber {
	return &OperationSubscriber{
		pool:   pool,
		log:    log,
		client: tendermint,
		query:  OpQueryIdentityStateTransfer,
	}
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

func (o *OperationSubscriber) Run(ctx context.Context) {
	o.log.Infof("[Pool] Subscribing to the pool. Query: %s", o.query)

	out, err := o.client.Subscribe(ctx, OpServiceName, o.query, OpPoolSize)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				if err := o.client.Unsubscribe(ctx, OpServiceName, o.query); err != nil {
					o.log.WithError(err).Error("[Pool] Failed to unsubscribe from new operations")
				}

				o.log.Info("Context finished")
				return
			case c, ok := <-out:
				if !ok {
					o.log.Fatal("[Pool] chanel closed")
				}

				for _, index := range c.Events[fmt.Sprintf("%s.%s", rarimo.EventTypeOperationApproved, rarimo.AttributeKeyOperationId)] {
					o.log.Infof("[Pool] New operation found index=%s", index)
					if err := o.pool.Add(index); err != nil {
						o.log.WithError(err).Error("error adding operation to the pool")
					}
				}
			}
		}
	}()
}
