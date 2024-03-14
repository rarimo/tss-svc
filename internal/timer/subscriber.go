package timer

import (
	"context"
	"time"

	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/logan/v3"
)

const (
	BlockServiceName = "block-subscriber"
	BlockQuery       = "tm.event = 'NewBlock'"
	ChanelCap        = 100
)

// BlockSubscriber subscribes to the NewBlock events on the tendermint core.
// New blocks indexes will be pushed to the timer and used in future for session timestamping
type BlockSubscriber struct {
	timer  *Timer
	client *http.HTTP
	log    *logan.Entry
}

// NewBlockSubscriber creates the subscriber instance for listening new blocks
func NewBlockSubscriber(timer *Timer, tendermint *http.HTTP, log *logan.Entry) *BlockSubscriber {
	return &BlockSubscriber{
		timer:  timer,
		log:    log,
		client: tendermint,
	}
}

func (b *BlockSubscriber) Run(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				b.log.Info("Context finished")
				return
			case <-ticker.C:
				info, err := b.client.Status(ctx)
				if err != nil {
					b.log.WithError(err).Fatal("[Block] failed to receive status")
				}

				b.log.Infof("[Block] Received New Block %s height: %d", info.SyncInfo.LatestBlockHash, info.SyncInfo.LatestBlockHeight)
				b.timer.newBlock(uint64(info.SyncInfo.LatestBlockHeight))

				ticker.Reset(5 * time.Second)
			}
		}
	}()
}
