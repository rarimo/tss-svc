package timer

import (
	"context"

	"github.com/tendermint/tendermint/rpc/client/http"
	"github.com/tendermint/tendermint/types"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

const (
	BlockServiceName = "block-subscriber"
	BlockQuery       = "tm.event = 'NewBlock'"
)

// BlockSubscriber subscribes to the NewBlock events on the tendermint core.
// New blocks indexes will be pushed to the timer and used in future for session timestamping
type BlockSubscriber struct {
	timer  *Timer
	client *http.HTTP
	log    *logan.Entry
}

func NewBlockSubscriber(timer *Timer, cfg config.Config) (*BlockSubscriber, error) {
	s := &BlockSubscriber{
		timer:  timer,
		log:    cfg.Log(),
		client: cfg.Tendermint(),
	}

	return s, s.subscribe()
}

func (b *BlockSubscriber) subscribe() error {
	out, err := b.client.Subscribe(context.Background(), BlockServiceName, BlockQuery)
	if err != nil {
		return err
	}

	go func() {
		for {
			c, ok := <-out
			if !ok {
				if err := b.client.Unsubscribe(context.Background(), BlockServiceName, BlockQuery); err != nil {
					b.log.WithError(err).Error("error unsubscribing from new blocks")
				}
				break
			}

			switch data := c.Data.(type) {
			case types.EventDataNewBlock:
				b.log.Infof("Received New Block %s height: %d \n", data.Block.Hash().String(), data.Block.Height)
				b.timer.newBlock(uint64(data.Block.Height))
				break
			}

		}
	}()

	return nil
}
