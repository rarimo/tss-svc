package connectors

import (
	"context"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// BroadcastConnector uses SubmitConnector to broadcast request to all parties, except of self.
// Is request submission fails, there will be ONE retry after last party submission.
type BroadcastConnector struct {
	params *local.Params
	*SubmitConnector
	log *logan.Entry
}

func NewBroadcastConnector(cfg config.Config) (*BroadcastConnector, error) {
	params, err := local.NewStorage(cfg)
	if err != nil {
		return nil, err
	}

	return &BroadcastConnector{
		params:          params,
		SubmitConnector: NewSubmitConnector(cfg),
		log:             cfg.Log(),
	}, nil
}

func (b *BroadcastConnector) SubmitAll(ctx context.Context, request *types.MsgSubmitRequest) {
	retry := b.submitAll(ctx, b.params.Parties(), request)
	b.submitAll(ctx, retry, request)
}

func (b *BroadcastConnector) submitAll(ctx context.Context, parties []*rarimo.Party, request *types.MsgSubmitRequest) []*rarimo.Party {
	failed := make([]*rarimo.Party, 0, b.params.N())

	for _, party := range parties {
		if party.PubKey != b.secret.ECDSAPubKeyStr() {
			_, err := b.Submit(ctx, *party, request)

			if err != nil {
				b.log.WithError(err).Errorf("error submitting request to party key: %s addr: %s", party.PubKey, party.Address)
				failed = append(failed, party)
			}
		}
	}

	return failed
}
