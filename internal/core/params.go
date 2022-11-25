package core

import (
	"context"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"google.golang.org/grpc"
)

type ParamsSnapshot struct {
	tssP   *rarimo.Params
	tokenP *token.Params
}

func NewParamsSnapshot(client *grpc.ClientConn) (*ParamsSnapshot, error) {
	tssP, err := rarimo.NewQueryClient(client).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	tokenP, err := token.NewQueryClient(client).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}
	return &ParamsSnapshot{
		tssP:   &tssP.Params,
		tokenP: &tokenP.Params,
	}, nil
}

func (p *ParamsSnapshot) TssParams() *rarimo.Params {
	return p.tssP
}

func (p *ParamsSnapshot) TokenParams() *token.Params {
	return p.tokenP
}

func (p *ParamsSnapshot) Parties() []*rarimo.Party {
	return p.tssP.Parties
}

func (p *ParamsSnapshot) Steps() []*rarimo.Step {
	return p.tssP.Steps
}

func (p *ParamsSnapshot) Step(id int) *rarimo.Step {
	return p.tssP.Steps[id]
}

func (p *ParamsSnapshot) N() int {
	return len(p.tssP.Parties)
}

func (p *ParamsSnapshot) T() int {
	return int(p.tssP.Threshold)
}

func (p *ParamsSnapshot) IsParty(key string) bool {
	for _, party := range p.tssP.Parties {
		if party.PubKey == key {
			return true
		}
	}

	return false
}

func (p *ParamsSnapshot) Party(key string) (rarimo.Party, bool) {
	for _, party := range p.tssP.Parties {
		if party.PubKey == key {
			return *party, true
		}
	}

	return rarimo.Party{}, false
}

func (p *ParamsSnapshot) ChainParams(network string) *token.ChainParams {
	return p.tokenP.Networks[network]
}
