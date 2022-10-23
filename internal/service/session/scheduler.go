package session

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
)

type Scheduler struct {
	currentSession uint64

	params *ParamsSaver
	timer  *Timer

	storage *pg.Storage
	log     *logan.Entry
}

func (s *Scheduler) receiveNextBlock(height uint64) error {
	return nil
}
