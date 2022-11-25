package controllers

import (
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"google.golang.org/grpc"
)

type defaultController struct {
	log       *logan.Entry
	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	params    *core.ParamsSnapshot
}

type ControllerFactory struct {
	def     *defaultController
	client  *grpc.ClientConn
	core    *connectors.CoreConnector
	storage secret.Storage
	pool    *pool.Pool
}

func NewControllerFactory(
	core *connectors.CoreConnector,
	storage secret.Storage,
	params *core.ParamsSnapshot,
	client *grpc.ClientConn,
	pool *pool.Pool,
	log *logan.Entry,
) *ControllerFactory {
	return &ControllerFactory{
		def: &defaultController{
			log:       log,
			broadcast: connectors.NewBroadcastConnector(params, connectors.NewSubmitConnector(storage.GetTssSecret()), log),
			auth:      core.NewRequestAuthorizer(params, log),
			params:    params,
		},
		client:  client,
		core:    core,
		storage: storage,
		pool:    pool,
	}
}

func (c *ControllerFactory) NewWithParams(params *core.ParamsSnapshot) *ControllerFactory {
	return &ControllerFactory{
		def: &defaultController{
			log:       c.def.log,
			broadcast: connectors.NewBroadcastConnector(params, connectors.NewSubmitConnector(c.storage.GetTssSecret()), c.def.log),
			auth:      core.NewRequestAuthorizer(params, c.def.log),
			params:    params,
		},
		client:  c.client,
		core:    c.core,
		storage: c.storage,
		pool:    c.pool,
	}
}
