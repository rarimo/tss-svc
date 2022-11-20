package local

import (
	"context"
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"google.golang.org/grpc"
)

// Params implements singleton pattern
var params *Params

// Params handles core global parameters
// and called up to be the source for the parameters in all components.
type Params struct {
	tssP    *rarimo.Params
	tokenP  *token.Params
	chainId string

	rarimo *grpc.ClientConn

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
			rarimo:     cfg.Cosmos(),
			nextTssP:   make(chan *rarimo.Params, 100),
			nextTokenP: make(chan *token.Params, 100),
		}
	}
	return params
}

func (p *Params) FetchParams() error {
	tssP, err := rarimo.NewQueryClient(p.rarimo).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		return err
	}

	tokenP, err := token.NewQueryClient(p.rarimo).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		return err
	}

	p.nextTssP <- &tssP.Params
	p.nextTokenP <- &tokenP.Params
	return nil
}

// UpdateParams checks and updates params if there are the new one
func (p *Params) UpdateParams() {
	for {
		select {
		case params := <-p.nextTssP:
			p.tssP = params
		case params := <-p.nextTokenP:
			p.tokenP = params
		default:
			return
		}
	}
}

func (p *Params) ChainId() string {
	return p.chainId
}

func (p *Params) TssParams() *rarimo.Params {
	return p.tssP
}

func (p *Params) TokenParams() *token.Params {
	return p.tokenP
}

func (p *Params) Parties() []*rarimo.Party {
	return p.tssP.Parties
}

func (p *Params) Steps() []*rarimo.Step {
	return p.tssP.Steps
}

func (p *Params) Step(id int) *rarimo.Step {
	return p.tssP.Steps[id]
}

func (p *Params) N() int {
	return len(p.tssP.Parties)
}

func (p *Params) T() int {
	return int(p.tssP.Threshold)
}

func (p *Params) IsParty(key string) bool {
	for _, party := range p.tssP.Parties {
		if party.PubKey == key {
			return true
		}
	}

	return false
}

func (p *Params) Party(key string) (rarimo.Party, bool) {
	for _, party := range p.tssP.Parties {
		if party.PubKey == key {
			return *party, true
		}
	}

	return rarimo.Party{}, false
}

func (p *Params) PartyByAccount(account string) (rarimo.Party, bool) {
	for _, party := range p.tssP.Parties {
		if party.Account == account {
			return *party, true
		}
	}

	return rarimo.Party{}, false
}

func (p *Params) ChainParams(network string) *token.ChainParams {
	return p.tokenP.Networks[network]
}

func (p *Params) PartyIds() tss.SortedPartyIDs {
	res := make([]*tss.PartyID, 0, len(p.tssP.Parties))

	for _, party := range p.tssP.Parties {
		_, data, err := bech32.DecodeAndConvert(party.Account)
		if err != nil {
			panic(err)
		}

		res = append(res, tss.NewPartyID(party.Account, "", new(big.Int).SetBytes(data)))
	}

	return tss.SortPartyIDs(res)
}
