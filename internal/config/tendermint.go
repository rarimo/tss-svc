package config

import (
	"github.com/tendermint/tendermint/rpc/client/http"
	_ "github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
)

type Tenderminter interface {
	Tendermint() *http.HTTP
}

type tenderminter struct {
	getter kv.Getter
	once   comfig.Once
}

func NewTenderminter(getter kv.Getter) Tenderminter {
	return &tenderminter{
		getter: getter,
	}
}

func (t *tenderminter) Tendermint() *http.HTTP {
	return t.once.Do(func() interface{} {
		var config struct {
			Addr string `fig:"addr"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(t.getter, "core")).Please(); err != nil {
			panic(err)
		}

		client, err := http.New(config.Addr, "/websocket")
		if err != nil {
			panic(err)
		}

		if err := client.Start(); err != nil {
			panic(err)
		}

		return client
	}).(*http.HTTP)
}
