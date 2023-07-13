package config

import (
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
)

type SwaggerInfo struct {
	Addr    string `fig:"addr"`
	Enabled bool   `fig:"enabled"`
}

func (c *config) Swagger() *SwaggerInfo {
	return c.swagger.Do(func() interface{} {
		info := &SwaggerInfo{}
		if err := figure.Out(info).From(kv.MustGetStringMap(c.getter, "swagger")).Please(); err != nil {
			panic(err)
		}
		return info
	}).(*SwaggerInfo)
}
