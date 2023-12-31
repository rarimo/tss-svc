package connectors

import (
	"context"
	"fmt"
	"sync"

	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"github.com/rarimo/tss-svc/internal/secret"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	retry := b.SubmitToWithReport(ctx, coreCon, request, b.parties...)
	b.SubmitToWithReport(ctx, coreCon, request, retry...)
}

// Deprecated: SubmitAll is deprecated. Use SubmitAllWithReport instead
func (b *BroadcastConnector) SubmitAll(ctx context.Context, request *types.MsgSubmitRequest) {
	retry := b.SubmitTo(ctx, request, b.parties...)
	b.SubmitTo(ctx, request, retry...)
}

func (b *BroadcastConnector) SubmitToWithReport(ctx context.Context, coreCon *CoreConnector, request *types.MsgSubmitRequest, parties ...*rarimo.Party) []*rarimo.Party {
	request.Data.SessionType = b.sessionType

	failed := struct {
		mu  sync.Mutex
		arr []*rarimo.Party
	}{
		arr: make([]*rarimo.Party, 0, len(b.parties)),
	}

	for _, party := range parties {
		if party.Account != b.sc.AccountAddress() {
			b.log.Debugf("Sending message to: %s, addr: %s", party.Account, party.Address)
			if _, err := b.Submit(ctx, party, request); err != nil {
				b.log.WithError(err).Errorf("Error submitting request to party: %s addr: %s", party.Account, party.Address)

				// check that party returned an error
				if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
					func() {
						failed.mu.Lock()
						defer failed.mu.Unlock()
						failed.arr = append(failed.arr, party)
					}()
					continue
				}

				if err := coreCon.SubmitReport(
					request.Data.Id,
					rarimo.ViolationType_Offline,
					party.Account,
					fmt.Sprintf("Party was offline when tried to submit %s request", request.Data.Type),
				); err != nil {
					b.log.WithError(err).Errorf("Error submitting violation report for party: %s", party.Account)
				}

				continue
			}

			b.log.Debugf("Successfully sent message to: %s, addr: %s", party.Account, party.Address)
		}
	}

	return failed.arr
}

// Deprecated: SubmitTo is deprecated. Use SubmitToWithReport instead
func (b *BroadcastConnector) SubmitTo(ctx context.Context, request *types.MsgSubmitRequest, parties ...*rarimo.Party) []*rarimo.Party {
	request.Data.SessionType = b.sessionType

	failed := struct {
		mu  sync.Mutex
		arr []*rarimo.Party
	}{
		arr: make([]*rarimo.Party, 0, len(b.parties)),
	}

	for _, party := range parties {
		if party.Account != b.sc.AccountAddress() {
			go func(to rarimo.Party) {
				if _, err := b.Submit(ctx, &to, request); err != nil {
					b.log.WithError(err).Errorf("Error submitting request to party: %s addr: %s", to.Account, to.Address)
					func() {
						failed.mu.Lock()
						defer failed.mu.Unlock()
						failed.arr = append(failed.arr, &to)
					}()
				}
			}(*party)
		}
	}

	return failed.arr
}
