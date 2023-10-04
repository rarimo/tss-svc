package config

import (
	"github.com/rarimo/tss-svc/internal/data/pg"
	"gitlab.com/distributed_lab/kit/pgdb"
)

func (c *config) Storage() *pg.Storage {
	return c.storage.Do(func() interface{} {
		return pg.New(pgdb.NewDatabaser(c.getter).DB())
	}).(*pg.Storage)
}
