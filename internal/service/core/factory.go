package core

import (
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/auth"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/connectors"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"google.golang.org/grpc"
)

type defaultController struct {
	*logan.Entry
	*connectors.ConfirmConnector
	*connectors.BroadcastConnector
	*Session
	auth    *auth.RequestAuthorizer
	rarimo  *grpc.ClientConn
	secret  *local.Secret
	params  *local.Params
	reshare *ReshareProvider
	rats    *RatCounter
}

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
			Session:            NewSession(cfg),
			auth:               auth.NewRequestAuthorizer(cfg),
			rarimo:             cfg.Cosmos(),
			secret:             local.NewSecret(cfg),
			params:             local.NewParams(cfg),
			reshare:            NewReshareProvider(cfg),
			rats:               NewRatCounter(cfg),
		},
		pool:     pool.NewPool(cfg),
		proposer: NewProposerProvider(cfg),
	}
}

func (*ControllerFactory) GetEmptyController(next IController, bounds *bounds) IController {
	return NewEmptyController(next, bounds)
}

func (c *ControllerFactory) GetProposalController(proposer rarimo.Party, bounds *bounds) IController {
	return NewProposalController(proposer, c.pool, c.defaultController, bounds, c)
}

func (c *ControllerFactory) GetAcceptanceController(data ProposalData, bounds *bounds) IController {
	return NewAcceptanceController(c.defaultController, data, bounds, c)
}

func (c *ControllerFactory) GetSignatureController(data AcceptanceData, bounds *bounds) IController {
	return NewSignatureController(data, bounds, c.defaultController, c)
}

func (c *ControllerFactory) GetFinishController(data SignatureData, bounds *bounds) IController {
	return NewFinishController(data, c.proposer, c.defaultController, bounds, c)
}

func (c *ControllerFactory) GetKeygenController(bounds *bounds) IController {
	return NewKeygenController(c.defaultController, bounds, c)
}

func (c *ControllerFactory) GetReshareController(data AcceptanceData, bounds *bounds) IController {
	return NewReshareController(data, c.defaultController, bounds, c)
}
