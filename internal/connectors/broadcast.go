package connectors

import (
	"context"
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// BroadcastConnector uses SubmitConnector to broadcast request to all parties, except of self.
// Is request submission fails, there will be ONE retry after last party submission.
type BroadcastConnector struct {
	*SubmitConnector
	sessionType types.SessionType
	parties     []*rarimo.Party
	sc          *secret.TssSecret
	log         *logan.Entry
}

func NewBroadcastConnector(sessionType types.SessionType, parties []*rarimo.Party, sc *secret.TssSecret, log *logan.Entry) *BroadcastConnector {
	return &BroadcastConnector{
		SubmitConnector: NewSubmitConnector(sc),
		sessionType:     sessionType,
		parties:         parties,
		sc:              sc,
		log:             log,
	}
}

func (b *BroadcastConnector) SubmitAllWithReport(ctx context.Context, coreCon *CoreConnector, request *types.MsgSubmitRequest) {
	retry := b.SubmitTo(ctx, request, b.parties...)
	for _, party := range b.SubmitTo(ctx, request, retry...) {
		go func(offender string) {
			if err := coreCon.SubmitReport(request.Id, rarimo.ViolationType_Offline, offender, ""); err != nil {
				b.log.WithError(err).Errorf("Error submitting violation report for party: %s", offender)
			}
		}(party.Account)
	}
}

func (b *BroadcastConnector) SubmitAll(ctx context.Context, request *types.MsgSubmitRequest) {
	retry := b.SubmitTo(ctx, request, b.parties...)
	b.SubmitTo(ctx, request, retry...)
}

func (b *BroadcastConnector) SubmitToWithReport(ctx context.Context, coreCon *CoreConnector, request *types.MsgSubmitRequest, parties ...*rarimo.Party) []*rarimo.Party {
	failed := b.SubmitTo(ctx, request, parties...)
	for _, party := range failed {
		go func(offender string) {
			if err := coreCon.SubmitReport(request.Id, rarimo.ViolationType_Offline, offender, ""); err != nil {
				b.log.WithError(err).Errorf("Error submitting violation report for party: %s", offender)
			}
		}(party.Account)
	}

	return failed
}

func (b *BroadcastConnector) SubmitTo(ctx context.Context, request *types.MsgSubmitRequest, parties ...*rarimo.Party) []*rarimo.Party {
	request.SessionType = b.sessionType

	failed := struct {
		mu  sync.Mutex
		arr []*rarimo.Party
	}{
		arr: make([]*rarimo.Party, 0, len(b.parties)),
	}

	for _, party := range parties {
		if party.Account != b.sc.AccountAddress() {
			to := *party
			go func() {
				if _, err := b.Submit(ctx, to, request); err != nil {
					b.log.WithError(err).Errorf("Error submitting request to party: %s addr: %s", to.Account, to.Address)

					func() {
						failed.mu.Lock()
						defer failed.mu.Unlock()
						failed.arr = append(failed.arr, &to)
					}()
				}
			}()
		}
	}

	return failed.arr
}
