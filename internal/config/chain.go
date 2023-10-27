package config

import (
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
)

type ChainParams struct {
	ChainId        string `fig:"chain_id"`
	CoinName       string `fig:"coin_name"`
	DisableReports bool   `fig:"disable_reports"`
}

func (c *config) ChainParams() *ChainParams {
	return c.chain.Do(func() interface{} {
		var params ChainParams

		if err := figure.Out(&params).From(kv.MustGetStringMap(c.getter, "chain")).Please(); err != nil {
			panic(err)
		}

		return &params
	}).(*ChainParams)
}
