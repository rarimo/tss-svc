package controllers

import (
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"google.golang.org/grpc"
)

type defaultController struct {
	log       *logan.Entry
	broadcast *connectors.BroadcastConnector
	auth      *core.RequestAuthorizer
	params    *core.ParamsSnapshot
}

type ControllerFactory struct {
	def      *defaultController
	client   *grpc.ClientConn
	core     *connectors.CoreConnector
	storage  secret.Storage
	pool     *pool.Pool
	proposer *core.Proposer
}

func NewControllerFactory(
	con *connectors.CoreConnector,
	storage secret.Storage,
	params *core.ParamsSnapshot,
	client *grpc.ClientConn,
	pool *pool.Pool,
	log *logan.Entry,
	proposer *core.Proposer,
) *ControllerFactory {
	return &ControllerFactory{
		def: &defaultController{
			log:       log,
			broadcast: connectors.NewBroadcastConnector(params, connectors.NewSubmitConnector(storage.GetTssSecret()), log),
			auth:      core.NewRequestAuthorizer(params, log),
			params:    params,
		},
		client:   client,
		core:     con,
		storage:  storage,
		pool:     pool,
		proposer: proposer,
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
		client:   c.client,
		core:     c.core,
		storage:  c.storage,
		pool:     c.pool,
		proposer: c.proposer.WithParams(params),
	}
}

func (c *ControllerFactory) GetProposalController(sessionId uint64, bounds *core.Bounds) IController {
	data := LocalSessionData{
		Proposer:  c.proposer.NextProposer(sessionId),
		SessionId: sessionId,
	}

	return &ProposalController{
		defaultController: c.def,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		bounds:            bounds,
		storage:           c.storage,
		client:            c.client,
		pool:              c.pool,
		factory:           c,
		result:            LocalProposalData{LocalSessionData: data},
	}
}

func (c *ControllerFactory) GetAcceptanceController(bounds *core.Bounds, data LocalProposalData) IController {
	return &AcceptanceController{
		defaultController: c.def,
		mu:                &sync.Mutex{},
		wg:                &sync.WaitGroup{},
		bounds:            bounds,
		data:              data,
		storage:           c.storage,
		factory:           c,
		result:            LocalAcceptanceData{LocalProposalData: data},
	}
}

func (c *ControllerFactory) GetSignController(bounds *core.Bounds, data LocalAcceptanceData) IController {
	return &SignatureController{
		defaultController: c.def,
		bounds:            bounds,
		data:              data,
		party:             tss.NewSignParty(data.Root, data.SessionId, c.storage, c.storage.GetTssSecret(), tss.NewPartiesSetData(c.def.params.Parties()), c.def.broadcast, c.def.log),
		factory:           c,
		result:            LocalSignatureData{LocalAcceptanceData: data},
	}
}

func (c *ControllerFactory) GetFinishController(bounds *core.Bounds, data LocalSignatureData) IController {
	return &FinishController{
		defaultController: c.def,
		wg:                &sync.WaitGroup{},
		bounds:            bounds,
		data:              data,
		core:              c.core,
		proposer:          c.proposer,
		factory:           c,
	}
}
