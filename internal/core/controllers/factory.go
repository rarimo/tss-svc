package controllers

import (
	"sync"

	"gitlab.com/rarimo/tss/tss-svc/internal/connectors"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/tss"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

// ControllerFactory is used to store current session data that will be used to create controllers inside controllers.
type ControllerFactory struct {
	data *LocalSessionData
}

// NewControllerFactory creates the new factory instance. The factory instance is associated with the certain session.
// That constructor should be executed only once during service launch.
// The input set that contains all necessary session information will be generated.
// Proposer also will be selected.
func NewControllerFactory(ctx core.Context, id uint64, sessionType types.SessionType) *ControllerFactory {
	set := core.NewInputSet(ctx.Client())
	core.SetInRegistry(
		core.ContextKeyBySessionType[sessionType],
		core.LogKey,
		ctx.Log().WithField("id", id).WithField("type", sessionType.String()),
	)

	return &ControllerFactory{
		data: &LocalSessionData{
			SessionType: sessionType,
			SessionId:   id,
			Set:         set,
			Acceptances: make(map[string]struct{}),
			Proposer:    GetProposer(set.Parties, set.LastSignature, id),
			Offenders:   make(map[string]struct{}),
		},
	}
}

// NextFactory creates the new factory instance for the next session.
// The input set that contains all necessary session information will be generated.
// Proposer also will be selected.
func (c *ControllerFactory) NextFactory(sessionType types.SessionType) *ControllerFactory {
	ctx := core.DefaultSessionContext(sessionType)
	core.SetInRegistry(
		core.ContextKeyBySessionType[sessionType],
		core.LogKey,
		ctx.Log().WithField("id", c.data.SessionId+1).WithField("type", sessionType.String()),
	)

	set := core.NewInputSet(ctx.Client())

	return &ControllerFactory{
		data: &LocalSessionData{
			SessionType: sessionType,
			SessionId:   c.data.SessionId + 1,
			Set:         set,
			Acceptances: make(map[string]struct{}),
			Proposer:    GetProposer(set.Parties, set.LastSignature, c.data.SessionId+1),
			Offenders:   make(map[string]struct{}),
		},
	}
}

// GetProposalController returns the proposal controller for the defined in the factory data session type.
// For types.SessionType_DefaultSession the proposal controller will be based on current parties set (all active and inactive parties).
// For types.SessionType_ReshareSession the proposal controller will be based on current parties set (all active and inactive parties).
func (c *ControllerFactory) GetProposalController() IController {
	ctx := core.DefaultSessionContext(c.data.SessionType)

	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &ProposalController{
			iProposalController: &defaultProposalController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
			},
			wg:      &sync.WaitGroup{},
			data:    c.data,
			auth:    core.NewRequestAuthorizer(c.data.Set.Parties, ctx.Log()),
			factory: c,
		}

	case types.SessionType_ReshareSession:
		return &ProposalController{
			iProposalController: &reshareProposalController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
			},
			wg:      &sync.WaitGroup{},
			data:    c.data,
			auth:    core.NewRequestAuthorizer(c.data.Set.Parties, ctx.Log()),
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
	ctx := core.DefaultSessionContext(c.data.SessionType)

	switch c.data.SessionType {
	case types.SessionType_DefaultSession:
		return &AcceptanceController{
			iAcceptanceController: &defaultAcceptanceController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
				factory:   c,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			auth: core.NewRequestAuthorizer(c.data.Set.Parties, ctx.Log()),
			log:  ctx.Log(),
		}
	case types.SessionType_ReshareSession:
		return &AcceptanceController{
			iAcceptanceController: &reshareAcceptanceController{
				data:      c.data,
				broadcast: connectors.NewBroadcastConnector(c.data.SessionType, c.data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
				factory:   c,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
			auth: core.NewRequestAuthorizer(c.data.Set.Parties, ctx.Log()),
		}
	}
	// Should not appear
	panic("Invalid session type")
}

// GetRootSignController returns the root signature controller based on the selected signers set.
func (c *ControllerFactory) GetRootSignController(hash string) IController {
	ctx := core.DefaultSessionContext(c.data.SessionType)

	parties := getSignersList(c.data.Signers, c.data.Set.Parties)
	return &SignatureController{
		iSignatureController: &rootSignatureController{
			data:    c.data,
			factory: c,
		},
		wg:    &sync.WaitGroup{},
		data:  c.data,
		auth:  core.NewRequestAuthorizer(parties, ctx.Log()),
		party: tss.NewSignParty(hash, c.data.SessionId, c.data.SessionType, parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
	}
}

// GetKeySignController returns the key signature controller based on the selected signers set.
func (c *ControllerFactory) GetKeySignController(hash string) IController {
	ctx := core.DefaultSessionContext(c.data.SessionType)

	parties := getSignersList(c.data.Signers, c.data.Set.Parties)
	return &SignatureController{
		iSignatureController: &keySignatureController{
			data:    c.data,
			factory: c,
		},
		wg:    &sync.WaitGroup{},
		data:  c.data,
		auth:  core.NewRequestAuthorizer(parties, ctx.Log()),
		party: tss.NewSignParty(hash, c.data.SessionId, c.data.SessionType, parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
	}
}

// GetFinishController returns the finish controller for the defined in the factory data session type.
func (c *ControllerFactory) GetFinishController() IController {
	switch c.data.SessionType {
	case types.SessionType_KeygenSession:
		return &FinishController{
			iFinishController: &keygenFinishController{
				data: c.data,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
		}
	case types.SessionType_ReshareSession:
		return &FinishController{
			iFinishController: &reshareFinishController{
				data: c.data,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
		}
	case types.SessionType_DefaultSession:
		return &FinishController{
			iFinishController: &defaultFinishController{
				data: c.data,
			},
			wg:   &sync.WaitGroup{},
			data: c.data,
		}
	}

	// Should not appear
	panic("Invalid session type")
}

// GetKeygenController returns the keygen controller for the defined in the factory data session type.
// For types.SessionType_KeygenSession the keygen controller will be based on current parties set (all parties should be inactive).
// For types.SessionType_ReshareSession the keygen controller will be based on current parties set (all active and inactive parties).
func (c *ControllerFactory) GetKeygenController() IController {
	ctx := core.DefaultSessionContext(c.data.SessionType)

	switch c.data.SessionType {
	case types.SessionType_ReshareSession:
		return &KeygenController{
			iKeygenController: &reshareKeygenController{
				data:    c.data,
				factory: c,
			},
			wg:    &sync.WaitGroup{},
			data:  c.data,
			auth:  core.NewRequestAuthorizer(c.data.Set.Parties, ctx.Log()),
			party: tss.NewKeygenParty(c.data.SessionId, c.data.SessionType, c.data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
		}
	case types.SessionType_KeygenSession:
		return &KeygenController{
			iKeygenController: &defaultKeygenController{
				data:    c.data,
				factory: c,
			},
			wg:    &sync.WaitGroup{},
			data:  c.data,
			auth:  core.NewRequestAuthorizer(c.data.Set.Parties, ctx.Log()),
			party: tss.NewKeygenParty(c.data.SessionId, c.data.SessionType, c.data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
		}
	}

	// Should not appear
	panic("Invalid session type")
}
