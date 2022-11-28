package core

import (
	"context"
	"crypto/ecdsa"
	"math/big"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	token "gitlab.com/rarify-protocol/rarimo-core/x/tokenmanager/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/secret"
	"google.golang.org/grpc"
)

type ParamsData struct {
	IsActive     bool
	GlobalPubKey string
	N, T         int
	Parties      []*rarimo.Party
	Chains       map[string]*token.ChainParams
}

type LocalData struct {
	LocalAccountAddress    string
	LocalAccountPrivateKey cryptotypes.PrivKey
	TrialPrivateKey        *ecdsa.PrivateKey
}

type LocalTss struct {
	LocalPubKey     string
	SortedPartyIDs  tss.SortedPartyIDs
	LocalPrivateKey *ecdsa.PrivateKey
	LocalData       *keygen.LocalPartySaveData
	LocalParams     *keygen.LocalPreParams
}

type InputSet struct {
	*ParamsData
	*LocalData
	*LocalTss
}

func NewInputSet(client *grpc.ClientConn, storage secret.Storage) *InputSet {
	tssP, err := rarimo.NewQueryClient(client).Params(context.TODO(), &rarimo.QueryParamsRequest{})
	if err != nil {
		panic(err)
	}

	tokenP, err := token.NewQueryClient(client).Params(context.TODO(), &token.QueryParamsRequest{})
	if err != nil {
		panic(err)
	}

	partyIds := make([]*tss.PartyID, 0, len(tssP.Params.Parties))
	for _, party := range tssP.Params.Parties {
		_, data, err := bech32.DecodeAndConvert(party.Account)
		if err != nil {
			panic(err)
		}

		partyIds = append(partyIds, tss.NewPartyID(party.Account, "", new(big.Int).SetBytes(data)))
	}

	return &InputSet{
		ParamsData: &ParamsData{
			IsActive:     !tssP.Params.IsUpdateRequired,
			GlobalPubKey: tssP.Params.KeyECDSA,
			N:            len(tssP.Params.Parties),
			T:            int(tssP.Params.Threshold),
			Parties:      tssP.Params.Parties,
			Chains:       tokenP.Params.Networks,
		},

		LocalData: &LocalData{
			LocalAccountAddress:    storage.AccountAddressStr(),
			LocalAccountPrivateKey: storage.AccountPrvKey(),
			TrialPrivateKey:        storage.GetTrialPrivateKey(),
		},

		LocalTss: &LocalTss{
			LocalPubKey:     storage.GetTssSecret().PubKeyStr(),
			SortedPartyIDs:  tss.SortPartyIDs(partyIds),
			LocalPrivateKey: storage.GetTssSecret().Prv,
			LocalData:       storage.GetTssSecret().Data,
			LocalParams:     storage.GetTssSecret().Params,
		},
	}
}

func (s *InputSet) Equals(other *InputSet) bool {
	if !PartiesEqual(s.Parties, other.Parties) {
		return false
	}

	if s.T != other.T {
		return false
	}

	return s.IsActive == other.IsActive
}

func (p *ParamsData) PartyByKey(key string) (rarimo.Party, bool) {
	for _, party := range p.Parties {
		if party.PubKey == key {
			return *party, true
		}
	}

	return rarimo.Party{}, false
}

func (p *ParamsData) PartyByAccount(acc string) (rarimo.Party, bool) {
	for _, party := range p.Parties {
		if party.Account == acc {
			return *party, true
		}
	}

	return rarimo.Party{}, false
}

func (l *LocalData) PartyKey() *big.Int {
	return new(big.Int).SetBytes(l.LocalAccountPrivateKey.PubKey().Address().Bytes())
}

func (s *InputSet) LocalParty() *tss.PartyID {
	return s.SortedPartyIDs.FindByKey(s.PartyKey())
}
