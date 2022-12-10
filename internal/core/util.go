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
	return p1.Account == p2.Account
}

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

func PartiesByAccountMapping(parties []*rarimo.Party) map[string]*rarimo.Party {
	pmap := make(map[string]*rarimo.Party)
	for _, p := range parties {
		pmap[p.Account] = p
	}
	return pmap
}

func GetProposer(parties []*rarimo.Party, sig string, sessionId uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(sig)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, sessionId)
	hash := eth.Keccak256(sigBytes, idBytes)
	return *parties[int(hash[len(hash)-1])%len(parties)]
}
