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
		switch p.Status {
		case rarimo.PartyStatus_Active:
			verifiedParties = append(verifiedParties, p)
		case rarimo.PartyStatus_Inactive:
			unverifiedParties = append(unverifiedParties, p)
		}
	}

	allParties := append(verifiedParties, unverifiedParties...)

	return &InputSet{
		IsActive:          !tssP.Params.IsUpdateRequired,
		GlobalPubKey:      tssP.Params.KeyECDSA,
		N:                 len(allParties),
		T:                 int(tssP.Params.Threshold),
		Parties:           allParties,
		VerifiedParties:   verifiedParties,
		UnverifiedParties: unverifiedParties,
		LastSignature:     tssP.Params.LastSignature,
	}
}
