package config

import (
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/kit/pgdb"
)

type Config interface {
	comfig.Logger
	comfig.Listenerer
	pgdb.Databaser
	Tenderminter
	Cosmoser
}

type config struct {
	comfig.Logger
	comfig.Listenerer
	pgdb.Databaser
	Tenderminter
	Cosmoser
	getter kv.Getter
}

func New(getter kv.Getter) Config {
	return &config{
		getter:       getter,
		Logger:       comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Listenerer:   comfig.NewListenerer(getter),
		Databaser:    pgdb.NewDatabaser(getter),
		Tenderminter: NewTenderminter(getter),
		Cosmoser:     NewCosmoser(getter),
	}
}
