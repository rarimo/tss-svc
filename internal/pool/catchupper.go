package pool

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"google.golang.org/grpc"
)

// OperationCatchupper catches up old unsigned operations from core.
type OperationCatchupper struct {
	pool   *Pool
	rarimo *grpc.ClientConn
	log    *logan.Entry
}

// NewOperationCatchupper creates the catchup instance for adding all unsigned operations to the pool
func NewOperationCatchupper(cfg config.Config) *OperationCatchupper {
	return &OperationCatchupper{
		pool:   NewPool(cfg),
		rarimo: cfg.Cosmos(),
		log:    cfg.Log(),
	}
}

func (o *OperationCatchupper) Run() {
	var nextKey []byte

	for {
		operations, err := rarimo.NewQueryClient(o.rarimo).OperationAll(context.TODO(), &rarimo.QueryAllOperationRequest{Pagination: &query.PageRequest{Key: nextKey}})
		if err != nil {
			panic(err)
		}

		for _, op := range operations.Operation {
			if op.Status != rarimo.OpStatus_INITIALIZED {
				o.log.Debug("[Pool] Operation is not INITIALIZED")
				continue
			}

			o.log.Infof("[Pool] New operation found index=%s", op.Index)
			err := o.pool.Add(op.Index)
			if err != nil {
				panic(err)
			}
		}

		nextKey = operations.Pagination.NextKey
		if nextKey == nil {
			return
		}
	}
}
