package controllers

import (
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/data/pg"
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
	pg       *pg.Storage
	log      *logan.Entry
}

func NewControllerFactory(cfg config.Config) *ControllerFactory {
	set := core.NewInputSet(cfg.Cosmos(), secret.NewLocalStorage(cfg))
	return &ControllerFactory{
		data: &LocalSessionData{
			SessionId:   cfg.Session().StartSessionId,
			Old:         set,
			New:         set,
			Acceptances: make(map[string]struct{}),
		},
		client:   cfg.Cosmos(),
		pool:     pool.NewPool(cfg),
		proposer: core.NewProposer(cfg).WithInputSet(set),
		storage:  secret.NewLocalStorage(cfg),
		pg:       cfg.Storage(),
		log:      cfg.Log(),
	}
}

func (c *ControllerFactory) NextFactory() *ControllerFactory {
	set := core.NewInputSet(c.client, c.storage)
	if c.data.New.Equals(c.data.Old) {
		c.log.Debug("Previous session old and new are equal")
		return &ControllerFactory{
			data: &LocalSessionData{
				SessionId:   c.data.SessionId + 1,
				Old:         c.data.New,
				New:         set,
				Acceptances: make(map[string]struct{}),
			},
			client:   c.client,
			pool:     c.pool,
			proposer: c.proposer.WithInputSet(set),
			storage:  c.storage,
			pg:       c.pg,
			log:      c.log,
		}
	}

	c.log.Debug("Previous session old and new are not equal. Setting up with previous session old and current new")
	return &ControllerFactory{
		data: &LocalSessionData{
			SessionId:   c.data.SessionId + 1,
			Old:         c.data.Old,
			New:         set,
			Acceptances: make(map[string]struct{}),
		},
		client:   c.client,
		pool:     c.pool,
		proposer: c.proposer.WithInputSet(set),
		storage:  c.storage,
		pg:       c.pg,
		log:      c.log,
	}
}

func (c *ControllerFactory) GetProposalController() IController {
	c.data.Proposer = c.proposer.NextProposer(c.data.SessionId)
	return &ProposalController{
		wg:        &sync.WaitGroup{},
		data:      c.data,
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		client:    c.client,
		pool:      c.pool,
		pg:        c.pg,
		factory:   c,
	}
}

func (c *ControllerFactory) GetDefaultAcceptanceController() IController {
	return &AcceptanceController{
		IAcceptanceController: &DefaultAcceptanceController{
			data:      c.data,
			broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
			log:       c.log,
			pg:        c.pg,
			factory:   c,
		},
		wg:   &sync.WaitGroup{},
		data: c.data,
		auth: core.NewRequestAuthorizer(c.data.New, c.log),
		log:  c.log,
	}
}

func (c *ControllerFactory) GetReshareAcceptanceController() IController {
	return &AcceptanceController{
		IAcceptanceController: &ReshareAcceptanceController{
			data:      c.data,
			broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
			log:       c.log,
			pg:        c.pg,
			factory:   c,
		},
		wg:   &sync.WaitGroup{},
		data: c.data,
		auth: core.NewRequestAuthorizer(c.data.New, c.log),
		log:  c.log,
	}
}

func (c *ControllerFactory) GetRootSignController(hash string) IController {
	return &SignatureController{
		ISignatureController: &RootSignatureController{
			data:    c.data,
			factory: c,
			pg:      c.pg,
			log:     c.log,
		},
		wg:    &sync.WaitGroup{},
		data:  c.data,
		auth:  core.NewRequestAuthorizer(c.data.Old, c.log),
		log:   c.log,
		party: tss.NewSignParty(hash, c.data.SessionId, c.data.AcceptedSigningPartyIds, c.data.Old, c.log),
	}
}

func (c *ControllerFactory) GetKeySignController(hash string) IController {
	return &SignatureController{
		ISignatureController: &KeySignatureController{
			data:    c.data,
			factory: c,
			pg:      c.pg,
			log:     c.log,
		},
		wg:    &sync.WaitGroup{},
		data:  c.data,
		auth:  core.NewRequestAuthorizer(c.data.Old, c.log),
		log:   c.log,
		party: tss.NewSignParty(hash, c.data.SessionId, c.data.AcceptedSigningPartyIds, c.data.Old, c.log),
	}
}

func (c *ControllerFactory) GetReshareController() IController {
	return &ReshareController{
		wg:      &sync.WaitGroup{},
		data:    c.data,
		auth:    core.NewRequestAuthorizer(c.data.New, c.log),
		log:     c.log,
		party:   tss.NewReshareParty(c.data.SessionId, c.data.Old, c.data.New, c.log),
		storage: c.storage,
		pg:      c.pg,
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
		pg:       c.pg,
		pool:     c.pool,
		factory:  c,
	}
}

func (c *ControllerFactory) GetKeygenController() IController {
	return &KeygenController{
		wg:      &sync.WaitGroup{},
		data:    c.data,
		auth:    core.NewRequestAuthorizer(c.data.New, c.log),
		log:     c.log,
		storage: c.storage,
		party:   tss.NewKeygenParty(c.data.SessionId, c.data.New, c.log),
		pg:      c.pg,
		factory: c,
	}
}
