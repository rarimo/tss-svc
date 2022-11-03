package generate

import (
	"math/big"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/binance-chain/tss-lib/tss"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
)

type Service struct {
}

func (s *Service) Run() {
	tss.SetCurve(secp256k1.S256())

	parties := tss.SortPartyIDs([]*tss.PartyID{})
	ctx := tss.NewPeerContext(parties)
	party := tss.NewPartyID("", "", new(big.Int))
	threshold := 0
	params := tss.NewParameters(ctx, party, len(parties), threshold)

	out := make(chan tss.Message)
	end := make(chan keygen.LocalPartySaveData)
	kgParty := keygen.NewLocalParty(params, out, end)
	kgParty.Start()

	msg := <-out

	msg.WireBytes()
}
