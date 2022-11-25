package old

import (
	"crypto/elliptic"
	"encoding/binary"
	goerr "errors"
	"math/big"

	"github.com/bnb-chain/tss-lib/crypto"
	"github.com/bnb-chain/tss-lib/tss"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/params"
)

type ProposerProvider struct {
	lastSig string
	params  *params.Params
}

func NewProposerProvider(cfg config.Config) *ProposerProvider {
	return &ProposerProvider{
		lastSig: cfg.Session().LastSignature,
		params:  params.NewParams(cfg),
	}
}

func (p *ProposerProvider) Update(sig string) {
	p.lastSig = sig
}

func (p *ProposerProvider) GetProposer(session uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(p.lastSig)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, session)
	hash := eth.Keccak256(sigBytes, idBytes)
	return *p.params.Parties()[int(hash[len(hash)-1])%p.params.N()]
}

var (
	ErrWrongSet = goerr.New("wrong parties set")
)

type ReshareProvider struct {
	reshare *rarimo.ChangeParties
	params  *params.Params
	NewKeys []string
	Key     string
}

func NewReshareProvider(cfg config.Config) *ReshareProvider {
	return &ReshareProvider{
		params: params.NewParams(cfg),
	}
}

func (r *ReshareProvider) Reshare(reshare *rarimo.ChangeParties) error {
	localParams := r.params.Parties()
	if len(localParams) != len(reshare.CurrentSet) {
		return ErrWrongSet
	}

	index := make(map[string]struct{})
	for _, p := range reshare.NewSet {
		index[p.Account] = struct{}{}
	}

	for i, p := range reshare.CurrentSet {
		// check that old sets are equal
		if p.Account != localParams[i].Account {
			return ErrWrongSet
		}

		// Only add operations supported now
		if _, ok := index[p.Account]; !ok {
			return ErrWrongSet
		}
	}

	return nil
}

func (r *ReshareProvider) Complete(index []*big.Int, keys []*crypto.ECPoint, key *crypto.ECPoint) {
	set := r.NewSet()
	res := make(map[string]string)
	r.NewKeys = make([]string, 0, len(keys))

	for i := range index {
		res[set.FindByKey(index[i]).Id] = hexutil.Encode(elliptic.Marshal(eth.S256(), keys[i].X(), keys[i].Y()))
	}

	for _, p := range r.reshare.NewSet {
		r.NewKeys = append(r.NewKeys, res[p.Account])
	}

	r.Key = hexutil.Encode(elliptic.Marshal(eth.S256(), key.X(), key.Y()))
}

func (r *ReshareProvider) NewSet() tss.SortedPartyIDs {
	if r.reshare == nil {
		return tss.SortedPartyIDs{}
	}
	return PartyIds(r.reshare.NewSet)
}

func (r *ReshareProvider) OldSet() tss.SortedPartyIDs {
	if r.reshare == nil {
		return tss.SortedPartyIDs{}
	}
	return PartyIds(r.reshare.CurrentSet)
}

func (r *ReshareProvider) NewT() int {
	return ((len(r.reshare.NewSet) + 2) / 3) * 2
}

func (r *ReshareProvider) OldT() int {
	return ((len(r.reshare.CurrentSet) + 2) / 3) * 2
}

func (r *ReshareProvider) NewN() int {
	return len(r.reshare.NewSet)
}

func (r *ReshareProvider) OldN() int {
	return len(r.reshare.CurrentSet)
}
