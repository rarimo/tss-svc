package local

import (
	"context"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

// Params implements singleton pattern
var params *Params

// Params handles core global parameters
// and called up to be the source for the parameters in all components.
type Params struct {
	tssP    *rarimo.Params
	tokenP  *token.Params
	chainId string

	nextTssP   chan *rarimo.Params
	nextTokenP chan *token.Params
}

func NewParams(cfg config.Config) *Params {
	if params == nil {
		tssP, err := rarimo.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &rarimo.QueryParamsRequest{})
		if err != nil {
			panic(err)
		}

		tokenP, err := token.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &token.QueryParamsRequest{})
		if err != nil {
			panic(err)
		}

		cfg.Log().Infof("fetched new tss params %v", tssP.Params)
		cfg.Log().Infof("fetched new token params %v", tokenP.Params)

		params = &Params{
			tssP:       &tssP.Params,
			tokenP:     &tokenP.Params,
			chainId:    cfg.Private().ChainId,
			nextTssP:   make(chan *rarimo.Params, 100),
			nextTokenP: make(chan *token.Params, 100),
		}
	}
	return params
}

// NewParams receives new parameters but does not update it until UpdateParams is called
func (s *Params) NewParams(tss *rarimo.Params, token *token.Params) {
	s.nextTssP <- tss
	s.nextTokenP <- token
}

// UpdateParams checks and updates params if there are the new one
func (s *Params) UpdateParams() {
	for {
		select {
		case p := <-s.nextTssP:
			s.tssP = p
		case p := <-s.nextTokenP:
			s.tokenP = p
		default:
			return
		}
	}
}

func (s *Params) ChainId() string {
	return s.chainId
}

func (s *Params) TssParams() *rarimo.Params {
	return s.tssP
}

func (s *Params) TokenParams() *token.Params {
	return s.tokenP
}

func (s *Params) Parties() []*rarimo.Party {
	return s.tssP.Parties
}

func (s *Params) Steps() []*rarimo.Step {
	return s.tssP.Steps
}

func (s *Params) Step(id int) *rarimo.Step {
	return s.tssP.Steps[id]
}

func (s *Params) N() int {
	return len(s.tssP.Parties)
}

func (s *Params) T() int {
	return int(s.tssP.Threshold)
}

func (s *Params) IsParty(key string) bool {
	for _, party := range s.Parties() {
		if party.PubKey == key {
			return true
		}
	}

	return false
}

func (s *Params) Party(key string) (rarimo.Party, bool) {
	for _, party := range s.Parties() {
		if party.PubKey == key {
			return *party, true
		}
	}

	return rarimo.Party{}, false
}

func (s *Params) ChainParams(network string) *token.ChainParams {
	return s.tokenP.Networks[network]
}
