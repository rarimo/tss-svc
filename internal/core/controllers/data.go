package controllers

import (
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"github.com/rarimo/tss-svc/internal/connectors"
	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/internal/secret"
	"github.com/rarimo/tss-svc/internal/tss"
	"github.com/rarimo/tss-svc/pkg/types"
)

// LocalSessionData represents all necessary data from current session to be shared between controllers.
type LocalSessionData struct {
	SessionId          uint64
	Processing         bool
	SessionType        types.SessionType
	Proposer           rarimo.Party
	Set                *core.InputSet
	NewSecret          *secret.TssSecret
	Indexes            []string
	Root               string
	Acceptances        map[string]struct{}
	OperationSignature string
	KeySignature       string
	NewParties         []*rarimo.Party
	Offenders          map[string]struct{}
	Signers            map[string]struct{}
	IsSigner           bool
}

func NewSessionData(ctx core.Context, id uint64, sessionType types.SessionType) *LocalSessionData {
	set := core.NewInputSet(ctx.Client())
	core.SetInRegistry(
		core.ContextKeyBySessionType[sessionType],
		core.LogKey,
		ctx.Log().WithField("id", id).WithField("type", sessionType.String()),
	)

	return &LocalSessionData{
		SessionType: sessionType,
		SessionId:   id,
		Set:         set,
		Acceptances: make(map[string]struct{}),
		Proposer:    GetProposer(set.Parties, set.LastSignature, id),
		Offenders:   make(map[string]struct{}),
	}
}

func (data *LocalSessionData) Next() *LocalSessionData {
	ctx := core.DefaultSessionContext(data.SessionType)
	core.SetInRegistry(
		core.ContextKeyBySessionType[data.SessionType],
		core.LogKey,
		ctx.Log().WithField("id", data.SessionId+1).WithField("type", data.SessionType.String()),
	)

	set := core.NewInputSet(ctx.Client())

	return &LocalSessionData{
		SessionType: data.SessionType,
		SessionId:   data.SessionId + 1,
		Set:         set,
		Acceptances: make(map[string]struct{}),
		Proposer:    GetProposer(set.Parties, set.LastSignature, data.SessionId+1),
		Offenders:   make(map[string]struct{}),
	}

}

// GetProposalController returns the proposal controller for the provided session data
// For types.SessionType_DefaultSession the proposal controller will be based on current parties set (all active and inactive parties).
// For types.SessionType_ReshareSession the proposal controller will be based on current parties set (all active and inactive parties).
func (data *LocalSessionData) GetProposalController() IController {
	ctx := core.DefaultSessionContext(data.SessionType)

	switch data.SessionType {
	case types.SessionType_DefaultSession:
		return &ProposalController{
			iProposalController: &defaultProposalController{
				data:      data,
				broadcast: connectors.NewBroadcastConnector(data.SessionType, data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
			},
			wg:   &sync.WaitGroup{},
			data: data,
			auth: core.NewRequestAuthorizer(data.Set.Parties, ctx.Log()),
		}

	case types.SessionType_ReshareSession:
		return &ProposalController{
			iProposalController: &reshareProposalController{
				data:      data,
				broadcast: connectors.NewBroadcastConnector(data.SessionType, data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
			},
			wg:   &sync.WaitGroup{},
			data: data,
			auth: core.NewRequestAuthorizer(data.Set.Parties, ctx.Log()),
		}
	}

	// Should not appear
	panic("Invalid session type")
}

// GetAcceptanceController returns the acceptance controller for the provided session data
// For types.SessionType_DefaultSession the acceptance controller will be based on current parties set (all active and inactive parties).
// For types.SessionType_ReshareSession the acceptance controller will be based on current parties set (all active and inactive parties).
func (data *LocalSessionData) GetAcceptanceController() IController {
	ctx := core.DefaultSessionContext(data.SessionType)

	switch data.SessionType {
	case types.SessionType_DefaultSession:
		return &AcceptanceController{
			iAcceptanceController: &defaultAcceptanceController{
				data:      data,
				broadcast: connectors.NewBroadcastConnector(data.SessionType, data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
			},
			wg:   &sync.WaitGroup{},
			data: data,
			auth: core.NewRequestAuthorizer(data.Set.Parties, ctx.Log()),
		}
	case types.SessionType_ReshareSession:
		return &AcceptanceController{
			iAcceptanceController: &reshareAcceptanceController{
				data:      data,
				broadcast: connectors.NewBroadcastConnector(data.SessionType, data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Log()),
			},
			wg:   &sync.WaitGroup{},
			data: data,
			auth: core.NewRequestAuthorizer(data.Set.Parties, ctx.Log()),
		}
	}
	// Should not appear
	panic("Invalid session type")
}

// GetRootSignController returns the root signature controller based on the selected signers set.
func (data *LocalSessionData) GetRootSignController() IController {
	ctx := core.DefaultSessionContext(data.SessionType)

	parties := getSignersList(data.Signers, data.Set.Parties)
	return &SignatureController{
		iSignatureController: &rootSignatureController{
			data: data,
		},
		wg:    &sync.WaitGroup{},
		data:  data,
		auth:  core.NewRequestAuthorizer(parties, ctx.Log()),
		party: tss.NewSignParty(data.Root, data.SessionId, data.SessionType, parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
	}
}

// GetKeySignController returns the key signature controller based on the selected signers set.
func (data *LocalSessionData) GetKeySignController() IController {
	hash := hexutil.Encode(eth.Keccak256(hexutil.MustDecode(data.NewSecret.GlobalPubKey())))
	ctx := core.DefaultSessionContext(data.SessionType)

	parties := getSignersList(data.Signers, data.Set.Parties)
	return &SignatureController{
		iSignatureController: &keySignatureController{
			data: data,
		},
		wg:    &sync.WaitGroup{},
		data:  data,
		auth:  core.NewRequestAuthorizer(parties, ctx.Log()),
		party: tss.NewSignParty(hash, data.SessionId, data.SessionType, parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
	}
}

// GetFinishController returns the finish controller for the provided session data
func (data *LocalSessionData) GetFinishController() IController {
	switch data.SessionType {
	case types.SessionType_KeygenSession:
		return &FinishController{
			iFinishController: &keygenFinishController{
				data: data,
			},
			wg:   &sync.WaitGroup{},
			data: data,
		}
	case types.SessionType_ReshareSession:
		return &FinishController{
			iFinishController: &reshareFinishController{
				data: data,
			},
			wg:   &sync.WaitGroup{},
			data: data,
		}
	case types.SessionType_DefaultSession:
		return &FinishController{
			iFinishController: &defaultFinishController{
				data: data,
			},
			wg:   &sync.WaitGroup{},
			data: data,
		}
	}

	// Should not appear
	panic("Invalid session type")
}

// GetKeygenController returns the keygen controller for the provided session data
// For types.SessionType_KeygenSession the keygen controller will be based on current parties set (all parties should be inactive).
// For types.SessionType_ReshareSession the keygen controller will be based on current parties set (all active and inactive parties).
func (data *LocalSessionData) GetKeygenController() IController {
	ctx := core.DefaultSessionContext(data.SessionType)

	switch data.SessionType {
	case types.SessionType_ReshareSession:
		return &KeygenController{
			iKeygenController: &reshareKeygenController{
				data: data,
			},
			wg:    &sync.WaitGroup{},
			data:  data,
			auth:  core.NewRequestAuthorizer(data.Set.Parties, ctx.Log()),
			party: tss.NewKeygenParty(data.SessionId, data.SessionType, data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
		}
	case types.SessionType_KeygenSession:
		return &KeygenController{
			iKeygenController: &defaultKeygenController{
				data: data,
			},
			wg:    &sync.WaitGroup{},
			data:  data,
			auth:  core.NewRequestAuthorizer(data.Set.Parties, ctx.Log()),
			party: tss.NewKeygenParty(data.SessionId, data.SessionType, data.Set.Parties, ctx.SecretStorage().GetTssSecret(), ctx.Core(), ctx.Log()),
		}
	}

	// Should not appear
	panic("Invalid session type")
}
