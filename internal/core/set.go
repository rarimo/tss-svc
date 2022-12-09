package core

import (
	"context"

	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"google.golang.org/grpc"
)

// InputSet defines data set (parties, params, etc.) to be used in session
type InputSet struct {
	IsActive          bool
	GlobalPubKey      string
	N, T              int
	Parties           []*rarimo.Party
	VerifiedParties   []*rarimo.Party
	UnverifiedParties []*rarimo.Party
	Chains            map[string]*token.ChainParams
	LastSignature     string
}

func NewInputSet(client *grpc.ClientConn) *InputSet {
	tssP, err := rarimo.NewQueryClient(client).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		panic(err)
	}

	tokenP, err := token.NewQueryClient(client).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		panic(err)
	}

	verifiedParties := make([]*rarimo.Party, 0, len(tssP.Params.Parties))
	unverifiedParties := make([]*rarimo.Party, 0, len(tssP.Params.Parties))
	for _, p := range tssP.Params.Parties {
		if p.Verified {
			unverifiedParties = append(unverifiedParties, p)
			continue
		}
		verifiedParties = append(verifiedParties, p)
	}

	return &InputSet{
		IsActive:          !tssP.Params.IsUpdateRequired,
		GlobalPubKey:      tssP.Params.KeyECDSA,
		N:                 len(tssP.Params.Parties),
		T:                 int(tssP.Params.Threshold),
		Parties:           tssP.Params.Parties,
		VerifiedParties:   verifiedParties,
		UnverifiedParties: unverifiedParties,
		Chains:            tokenP.Params.Networks,
		LastSignature:     tssP.Params.LastSignature,
	}
}
