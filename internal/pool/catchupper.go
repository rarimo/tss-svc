package pool

import (
	"context"
	"gitlab.com/distributed_lab/logan/v3"

	"github.com/cosmos/cosmos-sdk/types/query"
	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"google.golang.org/grpc"
)

var acceptableOperationTypes = map[rarimo.OpType]struct{}{
	rarimo.OpType_TRANSFER:                     {},
	rarimo.OpType_FEE_TOKEN_MANAGEMENT:         {},
	rarimo.OpType_IDENTITY_AGGREGATED_TRANSFER: {},
	rarimo.OpType_WORLDCOIN_IDENTITY_TRANSFER:  {},
	rarimo.OpType_CSCA_ROOT_UPDATE:             {},
	rarimo.OpType_IDENTITY_GIST_TRANSFER:       {},
	rarimo.OpType_IDENTITY_STATE_TRANSFER:      {},
	rarimo.OpType_PASSPORT_ROOT_UPDATE:         {},
	rarimo.OpType_ARBITRARY:                    {},
}

// OperationCatchupper catches up old unsigned operations from core.
type OperationCatchupper struct {
	pool   *Pool
	rarimo *grpc.ClientConn
	log    *logan.Entry
}

// NewOperationCatchupper creates the catchup instance for adding all unsigned operations to the pool
func NewOperationCatchupper(pool *Pool, core *grpc.ClientConn, log *logan.Entry) *OperationCatchupper {
	return &OperationCatchupper{
		pool:   pool,
		rarimo: core,
		log:    log,
	}
}

func (o *OperationCatchupper) Run(ctx context.Context) {
	var nextKey []byte

	for {
		select {
		case <-ctx.Done():
			o.log.Info("Context finished")
			return
		default:
			operations, err := rarimo.NewQueryClient(o.rarimo).OperationAll(ctx, &rarimo.QueryAllOperationRequest{Pagination: &query.PageRequest{Key: nextKey}})
			if err != nil {
				panic(err)
			}

			for _, op := range operations.Operation {
				if _, ok := acceptableOperationTypes[op.OperationType]; !ok {
					o.log.Debugf("[Pool] Operation %s has unsupported type for catchup", op.Index)
				}

				if op.Status != rarimo.OpStatus_APPROVED {
					o.log.Debugf("[Pool] Operation %s is not APPROVED", op.Index)
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
}
