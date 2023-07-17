package config

import (
	vault "github.com/hashicorp/vault/api"
	"github.com/tendermint/tendermint/rpc/client/http"
	"gitlab.com/distributed_lab/kit/comfig"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/kit/pgdb"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"google.golang.org/grpc"
)

type Config interface {
	comfig.Logger
	comfig.Listenerer
	pgdb.Databaser

	Tendermint() *http.HTTP
	Cosmos() *grpc.ClientConn
	Storage() *pg.Storage
	Session() *SessionInfo
	Vault() *vault.KVv2
	Swagger() *SwaggerInfo
	ChainParams() *ChainParams
}

type config struct {
	comfig.Logger
	comfig.Listenerer
	pgdb.Databaser

	tendermint comfig.Once
	cosmos     comfig.Once
	storage    comfig.Once
	session    comfig.Once
	private    comfig.Once
	vault      comfig.Once
	swagger    comfig.Once
	chain      comfig.Once

	getter kv.Getter
}

func New(getter kv.Getter) Config {
	return &config{
		getter:     getter,
		Logger:     comfig.NewLogger(getter, comfig.LoggerOpts{}),
		Listenerer: comfig.NewListenerer(getter),
		Databaser:  pgdb.NewDatabaser(getter),
	}
}
