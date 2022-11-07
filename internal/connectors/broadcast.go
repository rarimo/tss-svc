package connectors

import (
	"context"
	"time"

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

func NewBroadcastConnector(cfg config.Config) *BroadcastConnector {
	params := local.NewParams(cfg)
	return &BroadcastConnector{
		params:          params,
		SubmitConnector: NewSubmitConnector(cfg),
		log:             cfg.Log(),
	}
}

func (b *BroadcastConnector) SubmitAll(ctx context.Context, request *types.MsgSubmitRequest) {
	retry := b.SubmitTo(ctx, request, b.params.Parties()...)
	b.SubmitTo(ctx, request, retry...)
}

func (b *BroadcastConnector) SubmitTo(ctx context.Context, request *types.MsgSubmitRequest, parties ...*rarimo.Party) []*rarimo.Party {
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

func (b *BroadcastConnector) MustSubmitTo(ctx context.Context, request *types.MsgSubmitRequest, parties ...*rarimo.Party) {
	for _, party := range parties {
		retry = 0
		if party.PubKey != b.secret.ECDSAPubKeyStr() {
			for {
				if _, err := b.Submit(ctx, *party, request); err != nil {
					b.logErr(err)
					time.Sleep(time.Second)
					continue
				}
				break
			}
		}
	}
}

var retry = 0

// log every 10 retries
func (b *BroadcastConnector) logErr(err error) {
	retry++
	if retry%10 == 0 {
		b.log.Infof("Retry #%d", retry)
		b.log.WithError(err).Error("error sending request")
	}
}
