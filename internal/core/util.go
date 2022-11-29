package core

import (
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
)

func Equal(p1 *rarimo.Party, p2 *rarimo.Party) bool {
	return p1.Address == p2.Account
}

func PartiesEqual(p1 []*rarimo.Party, p2 []*rarimo.Party) bool {
	if len(p1) != len(p2) {
		return false
	}

	for i := range p1 {
		if p1[i].Address != p2[i].Address || p1[i].PubKey != p2[i].PubKey {
			return false
		}
	}
	return true
}

func PartyIds(parties []*rarimo.Party) tss.SortedPartyIDs {
	partyIds := make([]*tss.PartyID, 0, len(parties))
	for _, party := range parties {
		_, data, err := bech32.DecodeAndConvert(party.Account)
		if err != nil {
			panic(err)
		}

		partyIds = append(partyIds, tss.NewPartyID(party.Account, "", new(big.Int).SetBytes(data)))
	}

	return tss.SortPartyIDs(partyIds)
}
