package core

import (
	"context"

	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
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
	LastSignature     string
}

func NewInputSet(client *grpc.ClientConn) *InputSet {
	tssP, err := rarimo.NewQueryClient(client).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		panic(err)
	}

	verifiedParties := make([]*rarimo.Party, 0, len(tssP.Params.Parties))
	unverifiedParties := make([]*rarimo.Party, 0, len(tssP.Params.Parties))
	for _, p := range tssP.Params.Parties {
		if p.Verified {
			verifiedParties = append(verifiedParties, p)
			continue
		}
		unverifiedParties = append(unverifiedParties, p)
	}

	return &InputSet{
		IsActive:          !tssP.Params.IsUpdateRequired,
		GlobalPubKey:      tssP.Params.KeyECDSA,
		N:                 len(tssP.Params.Parties),
		T:                 int(tssP.Params.Threshold),
		Parties:           tssP.Params.Parties,
		VerifiedParties:   verifiedParties,
		UnverifiedParties: unverifiedParties,
		LastSignature:     tssP.Params.LastSignature,
	}
}
