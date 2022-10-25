package core

import (
	"context"

	"github.com/cosmos/cosmos-sdk/types/query"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"google.golang.org/grpc"
)

// OperationCatchupper - connector for catch upping old unsigned operations from core.
type OperationCatchupper struct {
	op     chan<- string
	rarimo *grpc.ClientConn
	log    *logan.Entry
}

func NewOperationCatchupper(op chan<- string, cfg config.Config) *OperationCatchupper {
	return &OperationCatchupper{
		op:     op,
		rarimo: cfg.Cosmos(),
		log:    cfg.Log(),
	}
}

// TODO provide catchup config

func (o *OperationCatchupper) Run() error {
	var nextKey []byte

	for {
		operations, err := rarimo.NewQueryClient(o.rarimo).OperationAll(context.TODO(), &rarimo.QueryAllOperationRequest{Pagination: &query.PageRequest{Key: nextKey}})
		if err != nil {
			return err
		}

		for _, op := range operations.Operation {
			if op.Signed {
				o.log.Debug("Operation already signed")
				continue
			}

			o.log.Infof("New operation found index=%s", op.Index)
			o.op <- op.Index
		}

		nextKey = operations.Pagination.NextKey
		if nextKey == nil {
			return nil
		}
	}
}
