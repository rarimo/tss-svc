package controllers

import (
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
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

func (c *ControllerFactory) NextFactory() *ControllerFactory {
	return &ControllerFactory{}
}

func (c *ControllerFactory) GetProposalController() IController {
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

func (c *ControllerFactory) GetSignRootController() IController {
	return &SignatureController{
		data:      c.data,
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		party:     nil,
		factory:   c,
	}
}

func (c *ControllerFactory) GetSignKeyController() IController {
	return &SignatureController{
		data:      c.data,
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		party:     nil,
		factory:   c,
	}
}

func (c *ControllerFactory) GetReshareController() IController {
	return &ReshareController{
		data:      c.data,
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		party:     nil,
		storage:   c.storage,
		factory:   c,
	}
}

func (c *ControllerFactory) GetFinishController() IController {
	return &FinishController{
		wg:        &sync.WaitGroup{},
		broadcast: connectors.NewBroadcastConnector(c.data.New, c.log),
		core:      connectors.NewCoreConnector(c.client, c.data.New.LocalData, c.log),
		auth:      core.NewRequestAuthorizer(c.data.New, c.log),
		log:       c.log,
		data:      c.data,
		proposer:  c.proposer,
		factory:   c,
	}
}
