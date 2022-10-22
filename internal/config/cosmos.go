package config

import (
	"time"

	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type Cosmoser interface {
	Cosmos() *grpc.ClientConn
}

type cosmoser struct {
	getter kv.Getter
	once   comfig.Once
}

func NewCosmoser(getter kv.Getter) Cosmoser {
	return &cosmoser{
		getter: getter,
	}
}

func (c *cosmoser) Cosmos() *grpc.ClientConn {
	return c.once.Do(func() interface{} {
		var config struct {
			Addr string `fig:"addr"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(c.getter, "cosmos")).Please(); err != nil {
			panic(err)
		}

		con, err := grpc.Dial(config.Addr, grpc.WithInsecure(), grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:    10 * time.Second, // wait time before ping if no activity
			Timeout: 20 * time.Second, // ping timeout
		}))
		if err != nil {
			panic(err)
		}

		return con
	}).(*grpc.ClientConn)
}
