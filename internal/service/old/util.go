package old

import (
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
)

func Equal(p1 *rarimo.Party, p2 *rarimo.Party) bool {
	return p1.Address == p2.Account
}

func PartyIds(parties []*rarimo.Party) tss.SortedPartyIDs {
	res := make([]*tss.PartyID, 0, len(parties))

	for _, party := range parties {
		_, data, err := bech32.DecodeAndConvert(party.Account)
		if err != nil {
			panic(err)
		}

		res = append(res, tss.NewPartyID(party.Account, "", new(big.Int).SetBytes(data)))
	}

	return tss.SortPartyIDs(res)
}
