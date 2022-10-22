package config

import (
	"github.com/tendermint/tendermint/rpc/client/http"
	_ "github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
)

func (c *config) Tendermint() *http.HTTP {
	return c.tendermint.Do(func() interface{} {
		var config struct {
			Addr string `fig:"addr"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(c.getter, "core")).Please(); err != nil {
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
