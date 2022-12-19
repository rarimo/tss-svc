package connectors

import (
	"context"
	"time"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// BroadcastConnector uses SubmitConnector to broadcast request to all parties, except of self.
// Is request submission fails, there will be ONE retry after last party submission.
type BroadcastConnector struct {
	*SubmitConnector
	parties []*rarimo.Party
	sc      *secret.TssSecret
	log     *logan.Entry
}

func NewBroadcastConnector(parties []*rarimo.Party, sc *secret.TssSecret, log *logan.Entry) *BroadcastConnector {
	return &BroadcastConnector{
		SubmitConnector: NewSubmitConnector(sc),
		parties:         parties,
		sc:              sc,
		log:             log,
	}
}

func (b *BroadcastConnector) SubmitAll(ctx context.Context, request *types.MsgSubmitRequest) {
	retry := b.SubmitTo(ctx, request, b.parties...)
	b.SubmitTo(ctx, request, retry...)
}

func (b *BroadcastConnector) SubmitTo(ctx context.Context, request *types.MsgSubmitRequest, parties ...*rarimo.Party) []*rarimo.Party {
	failed := make([]*rarimo.Party, 0, len(b.parties))

	for _, party := range parties {
		if party.Account != b.sc.AccountAddress() {
			_, err := b.Submit(ctx, *party, request)

			if err != nil {
				b.log.WithError(err).Errorf("Error submitting request to party: %s addr: %s", party.Account, party.Address)
				failed = append(failed, party)
			}
		}
	}

	return failed
}

func (b *BroadcastConnector) MustSubmitTo(ctx context.Context, request *types.MsgSubmitRequest, parties ...*rarimo.Party) {
	for _, party := range parties {
		if party.Account != b.sc.AccountAddress() {
			for {
				if _, err := b.Submit(ctx, *party, request); err != nil {
					b.log.WithError(err).Errorf("Error submitting request to the party %s", party.Account)
					time.Sleep(time.Second)
					continue
				}
				break
			}
		}
	}
}
