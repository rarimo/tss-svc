package local

import (
	"context"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

// Storage handles core global parameters
// and called up to be the source for the parameters in all components.
type Storage struct {
	tssP   *rarimo.Params
	tokenP *token.Params

	nextTssP   chan *rarimo.Params
	nextTokenP chan *token.Params
}

func NewStorage(cfg config.Config) (*Storage, error) {
	tssP, err := rarimo.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	tokenP, err := token.NewQueryClient(cfg.Cosmos()).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return &Storage{
		tssP:       &tssP.Params,
		tokenP:     &tokenP.Params,
		nextTssP:   make(chan *rarimo.Params, 100),
		nextTokenP: make(chan *token.Params, 100),
	}, nil
}

func (s *Storage) NewParams(tss *rarimo.Params, token *token.Params) {
	s.nextTssP <- tss
	s.nextTokenP <- token
}

func (s *Storage) UpdateParams() {
	for {
		select {
		case s.tssP = <-s.nextTssP:
		case s.tokenP = <-s.nextTokenP:
		default:
			return
		}
	}
}

func (s *Storage) TssParams() *rarimo.Params {
	return s.tssP
}

func (s *Storage) TokenParams() *token.Params {
	return s.tokenP
}

func (s *Storage) Parties() []*rarimo.Party {
	return s.tssP.Parties
}

func (s *Storage) Steps() []*rarimo.Step {
	return s.tssP.Steps
}

func (s *Storage) Step(id int) *rarimo.Step {
	return s.tssP.Steps[id]
}

func (s *Storage) N() int {
	return len(s.tssP.Parties)
}

func (s *Storage) T() int {
	// TODO
	return 0
}

func (s *Storage) IsParty(key string) bool {
	for _, party := range s.Parties() {
		if party.PubKey == key {
			return true
		}
	}

	return false
}

func (s *Storage) Party(key string) *rarimo.Party {
	for _, party := range s.Parties() {
		if party.PubKey == key {
			return party
		}
	}

	return nil
}

func (s *Storage) ChainParams(network string) *token.ChainParams {
	return s.tokenP.Networks[network]
}
