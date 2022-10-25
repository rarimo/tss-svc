package core

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

// BlockSubscriber - connector for subscribing to the NewBlock events on the tendermint core.
// New blocks indexes will be pushed to the uint64 chan and used in future for session timestamping
type BlockSubscriber struct {
	blocks chan<- uint64
	client *http.HTTP
	log    *logan.Entry
}

func NewBlockSubscriber(op chan<- uint64, cfg config.Config) (*BlockSubscriber, error) {
	s := &BlockSubscriber{
		blocks: op,
		log:    cfg.Log(),
		client: cfg.Tendermint(),
	}

	return s, s.subscribe()
}

func (b *BlockSubscriber) Close() error {
	close(b.blocks)
	return nil
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
				b.blocks <- uint64(data.Block.Height)
				break
			}

		}
	}()

	return nil
}
