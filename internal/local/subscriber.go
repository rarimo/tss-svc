package local

import (
	"context"
	"time"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

const (
	ServiceName = "params-subscriber"
	ParamsQuery = "tm.event='Tx' AND params_updated.params_update_type='CHANGE_SET'"
)

// ParamsSubscriber subscribes to the ParamsUpdated events on the tendermint core.
// New params will be pushed to the local.Params
type ParamsSubscriber struct {
	params *Params
	client *http.HTTP
	log    *logan.Entry
}

// NewParamsSubscriber creates the subscriber instance for listening new params
func NewParamsSubscriber(cfg config.Config) *ParamsSubscriber {
	return &ParamsSubscriber{
		params: NewParams(cfg),
		log:    cfg.Log(),
		client: cfg.Tendermint(),
	}
}

func (b *ParamsSubscriber) Run() {
	out, err := b.client.Subscribe(context.Background(), ServiceName, ParamsQuery)
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			_, ok := <-out
			if !ok {
				if err := b.client.Unsubscribe(context.Background(), ServiceName, ParamsQuery); err != nil {
					b.log.WithError(err).Error("[Params] Failed to unsubscribe from new params")
				}
				break
			}

			time.Sleep(5 * time.Second)
			b.log.Info("[Params] Received params updated event")
			if err := b.params.FetchParams(); err != nil {
				b.log.WithError(err).Error("[Params] Failed to fetch new params")
			}
		}
	}()
}
