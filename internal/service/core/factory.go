package core

import (
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/auth"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/service/pool"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

type ControllerFactory struct {
	defaultController *defaultController
	pool              *pool.Pool
	proposer          *ProposerProvider
}

func NewControllerFactory(cfg config.Config) *ControllerFactory {
	return &ControllerFactory{
		defaultController: &defaultController{
			Entry:              cfg.Log(),
			ConfirmConnector:   connectors.NewConfirmConnector(cfg),
			BroadcastConnector: connectors.NewBroadcastConnector(cfg),
			auth:               auth.NewRequestAuthorizer(cfg),
			rarimo:             cfg.Cosmos(),
			secret:             local.NewSecret(cfg),
			params:             local.NewParams(cfg),
		},
		pool:     pool.NewPool(cfg),
		proposer: NewProposerProvider(cfg),
	}
}

func (*ControllerFactory) GetEmptyController(next IController, bounds *bounds) IController {
	return NewEmptyController(next, bounds)
}

func (c *ControllerFactory) GetProposalController(sessionId uint64, proposer rarimo.Party, bounds *bounds) IController {
	return NewProposalController(sessionId, proposer, c.pool, c.defaultController, bounds, c)
}

func (c *ControllerFactory) GetAcceptanceController(sessionId uint64, data types.ProposalData, bounds *bounds) IController {
	return NewAcceptanceController(c.defaultController, sessionId, data, bounds, c)
}

func (c *ControllerFactory) GetSignatureController(sessionId uint64, data types.AcceptanceData, bounds *bounds) IController {
	return NewSignatureController(sessionId, data, bounds, c.defaultController, c)
}

func (c *ControllerFactory) GetFinishController(sessionId uint64, data types.SignatureData, bounds *bounds) IController {
	return NewFinishController(sessionId, data, c.proposer, c.defaultController, bounds, c)
}

func (c *ControllerFactory) GetKeygenController(bounds *bounds) IController {
	return NewKeygenController(c.defaultController, bounds, c)
}

func (c *ControllerFactory) GetReshareController(sessionId uint64, data types.AcceptanceData, bounds *bounds) IController {
	return NewReshareController(sessionId, data, c.defaultController, bounds, c)
}
