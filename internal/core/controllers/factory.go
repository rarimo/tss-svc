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
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

// ControllerFactory is used to store current session data that will be used to create controllers inside controllers.
type ControllerFactory struct {
	data    *LocalSessionData
	client  *grpc.ClientConn
	pool    *pool.Pool
	storage secret.Storage
	pg      *pg.Storage
	log     *logan.Entry
}

func NewControllerFactory(cfg config.Config) *ControllerFactory {
	set := core.NewInputSet(cfg.Cosmos())

	return &ControllerFactory{
		data: &LocalSessionData{
			SessionId:   cfg.Session().StartSessionId,
			Set:         set,
			Acceptances: make(map[string]struct{}),
			Secret:      secret.NewLocalStorage(cfg).GetTssSecret(),
			Proposer:    core.GetProposer(set.Parties, set.LastSignature, cfg.Session().StartSessionId),
		},
		client:  cfg.Cosmos(),
		pool:    pool.NewPool(cfg),
		storage: secret.NewLocalStorage(cfg),
		pg:      cfg.Storage(),
		log:     cfg.Log(),
	}
}

func (c *ControllerFactory) NextFactory() *ControllerFactory {
	set := core.NewInputSet(c.client)

	return &ControllerFactory{
		data: &LocalSessionData{
			SessionId:   c.data.SessionId + 1,
			Set:         set,
			Acceptances: make(map[string]struct{}),
			Secret:      c.storage.GetTssSecret(),
			Proposer:    core.GetProposer(set.VerifiedParties, set.LastSignature, c.data.SessionId+1),
		},
		client:  c.client,
		pool:    c.pool,
		storage: c.storage,
		pg:      c.pg,
		log:     c.log,
	}
}

func (c *ControllerFactory) GetProposalController() IController {
	if c.data.Set.IsActive {
		return &ProposalController{
			IProposalController: &DefaultProposalController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.Set.Parties, c.data.Secret, c.log),
				client:    c.client,
				pool:      c.pool,
				pg:        c.pg,
				log:       c.log,
			},
			wg:      &sync.WaitGroup{},
			data:    c.data,
			auth:    core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:     c.log,
			factory: c,
		}
	}

	return &ProposalController{
		IProposalController: &ReshareProposalController{
			data:      c.data,
			broadcast: connectors.NewBroadcastConnector(c.data.Set.Parties, c.data.Secret, c.log),
			pg:        c.pg,
			log:       c.log,
		},
		wg:      &sync.WaitGroup{},
		data:    c.data,
		auth:    core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
		log:     c.log,
		factory: c,
	}
}

func (c *ControllerFactory) GetAcceptanceController() IController {
	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &AcceptanceController{
			IAcceptanceController: &DefaultAcceptanceController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.Set.Parties, c.data.Secret, c.log),
				log:       c.log,
				pg:        c.pg,
				factory:   c,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			auth: core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:  c.log,
		}
	case types.SessionType_ReshareSession:
		return &AcceptanceController{
			IAcceptanceController: &ReshareAcceptanceController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.Set.Parties, c.data.Secret, c.log),
				log:       c.log,
				pg:        c.pg,
				factory:   c,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			auth: core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:  c.log,
		}
	}
	// Should not appear
	panic("Invalid session type")
}

func (c *ControllerFactory) GetRootSignController(hash string) IController {
	// Only verified parties that accepted request can sign
	parties := getPartiesAcceptances(c.data.Acceptances, c.data.Set.VerifiedParties)
	return &SignatureController{
		ISignatureController: &RootSignatureController{
			data:    c.data,
			factory: c,
			pg:      c.pg,
			log:     c.log,
		},
		wg:    &sync.WaitGroup{},
		data:  c.data,
		auth:  core.NewRequestAuthorizer(parties, c.log),
		log:   c.log,
		party: tss.NewSignParty(hash, c.data.SessionId, parties, c.data.Secret, c.log),
	}
}

func (c *ControllerFactory) GetKeySignController(hash string) IController {
	// Only verified parties that accepted request can sign
	parties := getPartiesAcceptances(c.data.Acceptances, c.data.Set.VerifiedParties)
	return &SignatureController{
		ISignatureController: &KeySignatureController{
			data:    c.data,
			factory: c,
			pg:      c.pg,
			log:     c.log,
		},
		wg:    &sync.WaitGroup{},
		data:  c.data,
		auth:  core.NewRequestAuthorizer(parties, c.log),
		log:   c.log,
		party: tss.NewSignParty(hash, c.data.SessionId, parties, c.data.Secret, c.log),
	}
}

func (c *ControllerFactory) GetFinishController() IController {
	switch c.data.SessionType {
	case types.SessionType_KeygenSession:
		return &FinishController{
			IFinishController: &KeygenFinishController{
				data:    c.data,
				storage: c.storage,
				core:    connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				log:     c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			pg:   c.pg,
			log:  c.log,
		}
	case types.SessionType_ReshareSession:
		return &FinishController{
			IFinishController: &ReshareFinishController{
				data:    c.data,
				storage: c.storage,
				core:    connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				log:     c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			pg:   c.pg,
			log:  c.log,
		}
	case types.SessionType_DefaultSession:
		return &FinishController{
			IFinishController: &DefaultFinishController{
				data: c.data,
				pool: c.pool,
				core: connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				log:  c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			pg:   c.pg,
			log:  c.log,
		}
	}
	return nil
}

func (c *ControllerFactory) GetKeygenController() IController {
	switch c.data.SessionType {
	case types.SessionType_ReshareSession:
		return &KeygenController{
			IKeygenController: &ReshareKeygenController{
				data:    c.data,
				pg:      c.pg,
				log:     c.log,
				factory: c,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			auth: core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:  c.log,
		}
	default:
		return &KeygenController{
			IKeygenController: &DefaultKeygenController{
				data:    c.data,
				pg:      c.pg,
				log:     c.log,
				factory: c,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			auth: core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
		}
	}
}
