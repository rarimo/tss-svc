package controllers

import (
	"sync"

	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/data/pg"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"gitlab.com/rarimo/tss/tss-svc/internal/tss"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/grpc"
)

// ControllerFactory is used to store current session data that will be used to create controllers inside controllers.
type ControllerFactory struct {
	data    *LocalSessionData
	client  *grpc.ClientConn
	storage secret.Storage
	pg      *pg.Storage
	log     *logan.Entry
}

func NewControllerFactory(cfg config.Config, id uint64, sessionType types.SessionType) *ControllerFactory {
	set := core.NewInputSet(cfg.Cosmos())
	return &ControllerFactory{
		data: &LocalSessionData{
			SessionType: sessionType,
			SessionId:   id,
			Set:         set,
			Acceptances: make(map[string]struct{}),
			Secret:      secret.NewVaultStorage(cfg).GetTssSecret(),
			Proposer:    GetProposer(set.Parties, set.LastSignature, id),
		},
		client:  cfg.Cosmos(),
		storage: secret.NewVaultStorage(cfg),
		pg:      cfg.Storage(),
		log:     cfg.Log().WithField("id", id).WithField("type", sessionType.String()),
	}
}

func (c *ControllerFactory) NextFactory(sessionType types.SessionType) *ControllerFactory {
	set := core.NewInputSet(c.client)

	return &ControllerFactory{
		data: &LocalSessionData{
			SessionType: sessionType,
			SessionId:   c.data.SessionId + 1,
			Set:         set,
			Acceptances: make(map[string]struct{}),
			Secret:      c.storage.GetTssSecret(),
			Proposer:    GetProposer(set.Parties, set.LastSignature, c.data.SessionId+1),
		},
		client:  c.client,
		storage: c.storage,
		pg:      c.pg,
		log:     c.log.WithField("id", c.data.SessionId+1).WithField("type", sessionType.String()),
	}
}

func (c *ControllerFactory) GetProposalController() IController {
	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &ProposalController{
			IProposalController: &DefaultProposalController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, c.data.Secret, c.log),
				core:      connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				client:    c.client,
				pg:        c.pg,
				log:       c.log,
			},
			wg:      &sync.WaitGroup{},
			data:    c.data,
			auth:    core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:     c.log,
			factory: c,
		}

	case types.SessionType_ReshareSession:
		return &ProposalController{
			IProposalController: &ReshareProposalController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, c.data.Secret, c.log),
				core:      connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
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

	// Should not appear
	panic("Invalid session type")
}

func (c *ControllerFactory) GetAcceptanceController() IController {
	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &AcceptanceController{
			IAcceptanceController: &DefaultAcceptanceController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, c.data.Secret, c.log),
				core:      connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
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
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, c.data.Secret, c.log),
				core:      connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
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
		party: tss.NewSignParty(hash, c.data.SessionId, c.data.SessionType, parties, c.data.Secret, c.client, c.log),
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
		party: tss.NewSignParty(hash, c.data.SessionId, c.data.SessionType, parties, c.data.Secret, c.client, c.log),
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
				pg:      c.pg,
				log:     c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			log:  c.log,
		}
	case types.SessionType_ReshareSession:
		return &FinishController{
			IFinishController: &ReshareFinishController{
				data:    c.data,
				storage: c.storage,
				core:    connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				pg:      c.pg,
				log:     c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			log:  c.log,
		}
	case types.SessionType_DefaultSession:
		return &FinishController{
			IFinishController: &DefaultFinishController{
				data: c.data,
				core: connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				log:  c.log,
				pg:   c.pg,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			log:  c.log,
		}
	}

	// Should not appear
	panic("Invalid session type")
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
			wg:    &sync.WaitGroup{},
			data:  c.data,
			auth:  core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:   c.log,
			party: tss.NewKeygenParty(c.data.SessionId, c.data.SessionType, c.data.Set.Parties, c.data.Secret, c.client, c.log),
		}
	case types.SessionType_KeygenSession:
		return &KeygenController{
			IKeygenController: &DefaultKeygenController{
				data:    c.data,
				pg:      c.pg,
				log:     c.log,
				factory: c,
			},
			wg:    &sync.WaitGroup{},
			data:  c.data,
			auth:  core.NewRequestAuthorizer(c.data.Set.Parties, c.log),
			log:   c.log,
			party: tss.NewKeygenParty(c.data.SessionId, c.data.SessionType, c.data.Set.Parties, c.data.Secret, c.client, c.log),
		}
	}

	// Should not appear
	panic("Invalid session type")
}
