package timer

import (
	"context"

	"github.com/tendermint/tendermint/rpc/client/http"
	"github.com/tendermint/tendermint/types"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
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

// NewBlockSubscriber creates the subscriber instance for listening new blocks
func NewBlockSubscriber(cfg config.Config) *BlockSubscriber {
	return &BlockSubscriber{
		timer:  NewTimer(cfg),
		log:    cfg.Log(),
		client: cfg.Tendermint(),
	}
}

func (b *BlockSubscriber) Run() {
	go func() {
		for {
			b.runner()
			b.log.Info("[Block] Resubscribing to the blocks...")
		}
	}()
}

func (b *BlockSubscriber) runner() {
	out, err := b.client.Subscribe(context.Background(), BlockServiceName, BlockQuery)
	if err != nil {
		panic(err)
	}

	for {
		c, ok := <-out
		if !ok {
			if err := b.client.Unsubscribe(context.Background(), BlockServiceName, BlockQuery); err != nil {
				b.log.WithError(err).Error("[Block] failed to unsubscribe from new blocks")
			}
			break
		}

		switch data := c.Data.(type) {
		case types.EventDataNewBlock:
			b.log.Infof("[Block] Received New Block %s height: %d", data.Block.Hash().String(), data.Block.Height)
			b.timer.newBlock(uint64(data.Block.Height))
			break
		}

	}
}