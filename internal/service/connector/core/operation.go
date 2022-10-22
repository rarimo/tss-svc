package core

import (
	"context"
	"fmt"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"google.golang.org/grpc"
)

// TODO also listening other operations
const (
	OpServiceName = "op-subscriber"
	OpQuery       = "tm.event='Tx' AND new_operation.operation_type='TRANSFER,CHANGE_KEY'"
	OpPoolSize    = 1000
)

// OperationSubscriber - connector for subscribing to the NewOperation events on the tendermint core.
// New blocks indexes will be pushed to the uint64 chan and used in future for session timestamping
type OperationSubscriber struct {
	op       chan<- rarimo.Operation
	client   *http.HTTP
	rarimo   *grpc.ClientConn
	log      *logan.Entry
	isClosed bool
}

func NewOperationSubscriber(op chan<- rarimo.Operation, cfg config.Config) (*OperationSubscriber, error) {
	s := &OperationSubscriber{
		op:       op,
		isClosed: false,
		log:      cfg.Log(),
		client:   cfg.Tendermint(),
		rarimo:   cfg.Cosmos(),
	}

	return s, s.subscribe()
}

func (o *OperationSubscriber) Close() error {
	close(o.op)
	return nil
}
func (o *OperationSubscriber) subscribe() error {
	out, err := o.client.Subscribe(context.Background(), OpServiceName, OpQuery, OpPoolSize)
	if err != nil {
		return err
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

				op, err := rarimo.NewQueryClient(o.rarimo).Operation(context.Background(), &rarimo.QueryGetOperationRequest{Index: index})
				if err != nil {
					o.log.WithError(err).Error("error getting operation entry")
					continue
				}

				o.op <- op.Operation
			}

		}
	}()

	return nil
}
