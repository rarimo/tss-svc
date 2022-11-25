package core

import (
	"encoding/binary"
	"math/big"

	"github.com/bnb-chain/tss-lib/tss"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
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

func NextProposer(parties []*rarimo.Party, signature string, sessionId uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(signature)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, sessionId)
	hash := eth.Keccak256(sigBytes, idBytes)
	return *parties[int(hash[len(hash)-1])%len(parties)]
}
