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

// NewControllerFactory creates the new factory instance. The factory instance is associated with the certain session.
// That constructor should be executed only once during service launch.
// The input set that contains all necessary session information will be generated.
// Proposer also will be selected.
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
			Offenders:   make(map[string]struct{}),
		},
		client:  cfg.Cosmos(),
		storage: secret.NewVaultStorage(cfg),
		pg:      cfg.Storage(),
		log:     cfg.Log().WithField("id", id).WithField("type", sessionType.String()),
	}
}

// NextFactory creates the new factory instance for the next session.
// The input set that contains all necessary session information will be generated.
// Proposer also will be selected.
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
			Offenders:   make(map[string]struct{}),
		},
		client:  c.client,
		storage: c.storage,
		pg:      c.pg,
		log:     c.log.WithField("id", c.data.SessionId+1).WithField("type", sessionType.String()),
	}
}

// GetProposalController returns the proposal controller for the defined in the factory data session type.
// For types.SessionType_DefaultSession the proposal controller will be based on current parties set (all active and inactive parties).
// For types.SessionType_ReshareSession the proposal controller will be based on current parties set (all active and inactive parties).
func (c *ControllerFactory) GetProposalController() IController {
	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &ProposalController{
			iProposalController: &defaultProposalController{
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
			iProposalController: &reshareProposalController{
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

// GetAcceptanceController returns the acceptance controller for the defined in the factory data session type.
// For types.SessionType_DefaultSession the acceptance controller will be based on current parties set (all active and inactive parties).
// For types.SessionType_ReshareSession the acceptance controller will be based on current parties set (all active and inactive parties).
func (c *ControllerFactory) GetAcceptanceController() IController {
	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &AcceptanceController{
			iAcceptanceController: &defaultAcceptanceController{
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
			iAcceptanceController: &reshareAcceptanceController{
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

// GetRootSignController returns the root signature controller based on the selected signers set.
func (c *ControllerFactory) GetRootSignController(hash string) IController {
	parties := getSignersList(c.data.Signers, c.data.Set.Parties)
	return &SignatureController{
		iSignatureController: &rootSignatureController{
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

// GetKeySignController returns the key signature controller based on the selected signers set.
func (c *ControllerFactory) GetKeySignController(hash string) IController {
	parties := getSignersList(c.data.Signers, c.data.Set.Parties)
	return &SignatureController{
		iSignatureController: &keySignatureController{
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

// GetFinishController returns the finish controller for the defined in the factory data session type.
func (c *ControllerFactory) GetFinishController() IController {
	switch c.data.SessionType {
	case types.SessionType_KeygenSession:
		return &FinishController{
			iFinishController: &keygenFinishController{
				data:    c.data,
				storage: c.storage,
				core:    connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				pg:      c.pg,
				log:     c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			core: connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
			log:  c.log,
		}
	case types.SessionType_ReshareSession:
		return &FinishController{
			iFinishController: &reshareFinishController{
				data:    c.data,
				storage: c.storage,
				core:    connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				pg:      c.pg,
				log:     c.log,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			core: connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
			log:  c.log,
		}
	case types.SessionType_DefaultSession:
		return &FinishController{
			iFinishController: &defaultFinishController{
				data: c.data,
				core: connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
				log:  c.log,
				pg:   c.pg,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			core: connectors.NewCoreConnector(c.client, c.storage.GetTssSecret(), c.log),
			log:  c.log,
		}
	}

	// Should not appear
	panic("Invalid session type")
}

// GetKeygenController returns the keygen controller for the defined in the factory data session type.
// For types.SessionType_KeygenSession the keygen controller will be based on current parties set (all parties should be inactive).
// For types.SessionType_ReshareSession the keygen controller will be based on current parties set (all active and inactive parties).
func (c *ControllerFactory) GetKeygenController() IController {
	switch c.data.SessionType {
	case types.SessionType_ReshareSession:
		return &KeygenController{
			iKeygenController: &reshareKeygenController{
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
			iKeygenController: &defaultKeygenController{
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
