package controllers

import (
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"gitlab.com/rarify-protocol/tss-svc/internal/tss"
	"google.golang.org/grpc"
)

type ControllerFactory struct {
	data     *LocalSessionData
	client   *grpc.ClientConn
	pool     *pool.Pool
	proposer *core.Proposer
	storage  secret.Storage
	log      *logan.Entry
}

func NewControllerFactory(cfg config.Config) *ControllerFactory {
	set := core.NewInputSet(cfg.Cosmos(), secret.NewLocalStorage(cfg))
	return &ControllerFactory{
		data: &LocalSessionData{
			SessionId: cfg.Session().StartSessionId,
			Old:       set,
			New:       set,
		},
		client:   cfg.Cosmos(),
		pool:     pool.NewPool(cfg),
		proposer: core.NewProposer(cfg).WithInputSet(set),
		storage:  secret.NewLocalStorage(cfg),
		log:      cfg.Log(),
	}
}

func (c *ControllerFactory) NextFactory() *ControllerFactory {
	set := core.NewInputSet(c.client, c.storage)
	if c.data.New.Equals(c.data.Old) {
		return &ControllerFactory{
			data: &LocalSessionData{
				SessionId: c.data.SessionId + 1,
				Old:       c.data.New,
				New:       set,
			},
			client:   c.client,
			pool:     c.pool,
			proposer: c.proposer.WithInputSet(set),
			storage:  c.storage,
			log:      c.log,
		}
	}

	if c.data.New.Equals(set) {
		return &ControllerFactory{
			data: &LocalSessionData{
				SessionId: c.data.SessionId + 1,
				Old:       set,
				New:       set,
			},
			client:   c.client,
			pool:     c.pool,
			proposer: c.proposer.WithInputSet(set),
			storage:  c.storage,
			log:      c.log,
		}
	}

	return &ControllerFactory{
		data: &LocalSessionData{
			SessionId: c.data.SessionId + 1,
			Old:       c.data.Old,
			New:       set,
		},
		client:   c.client,
		pool:     c.pool,
		proposer: c.proposer.WithInputSet(set),
		storage:  c.storage,
		log:      c.log,
	}
}

func (c *ControllerFactory) GetProposalController() IController {
	c.data.Proposer = c.proposer.NextProposer(c.data.SessionId)
	return &ProposalController{
		mu:        &sync.Mutex{},
		wg:        &sync.WaitGroup{},
		data:      c.data,
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		client:    c.client,
		pool:      c.pool,
		factory:   c,
	}
}

func (c *ControllerFactory) GetAcceptanceController() IController {
	return &AcceptanceController{
		mu:        &sync.Mutex{},
		wg:        &sync.WaitGroup{},
		data:      c.data,
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		factory:   c,
	}
}

// Used always for old params
// if it's a default session there is no difference
// otherwise we should sign with old keys

func (c *ControllerFactory) GetSignController(hash string) IController {
	return &SignatureController{
		mu:      &sync.Mutex{},
		data:    c.data,
		auth:    core.NewRequestAuthorizer(c.data.Old, c.log),
		log:     c.log,
		party:   tss.NewSignParty(hash, c.data.SessionId, c.data.Old, c.log),
		factory: c,
	}
}

func (c *ControllerFactory) GetReshareController() IController {
	return &ReshareController{
		mu:      &sync.Mutex{},
		data:    c.data,
		auth:    core.NewRequestAuthorizer(c.data.New, c.log),
		log:     c.log,
		party:   tss.NewReshareParty(c.data.Old, c.data.New, c.log),
		storage: c.storage,
		factory: c,
	}
}

func (c *ControllerFactory) GetFinishController() IController {
	return &FinishController{
		wg:       &sync.WaitGroup{},
		core:     connectors.NewCoreConnector(c.client, c.data.New.LocalData, c.log),
		log:      c.log,
		data:     c.data,
		proposer: c.proposer,
		factory:  c,
	}
}

func (c *ControllerFactory) GetKeygenController() IController {
	return &KeygenController{
		mu:      &sync.Mutex{},
		data:    c.data,
		auth:    core.NewRequestAuthorizer(c.data.New, c.log),
		log:     c.log,
		storage: c.storage,
		party:   tss.NewKeygenParty(c.data.New, c.log),
		factory: nil,
	}
}
