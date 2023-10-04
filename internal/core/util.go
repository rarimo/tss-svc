package core

import (
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
)

func GetTssPartyKey(account string) *big.Int {
	_, data, err := bech32.DecodeAndConvert(account)
	if err != nil {
		panic(err)
	}
	return new(big.Int).SetBytes(data)
}

func PartyIds(parties []*rarimo.Party) tss.SortedPartyIDs {
	partyIds := make([]*tss.PartyID, 0, len(parties))
	for _, party := range parties {
		partyIds = append(partyIds, tss.NewPartyID(party.Account, party.Account, GetTssPartyKey(party.Account)))
	}

	return tss.SortPartyIDs(partyIds)
}
