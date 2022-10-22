package config

import (
	"gitlab.com/distributed_lab/kit/pgdb"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
)

func (c *config) Storage() *pg.Storage {
	return c.storage.Do(func() interface{} {
		return pg.New(pgdb.NewDatabaser(c.getter).DB())
	}).(*pg.Storage)
}
