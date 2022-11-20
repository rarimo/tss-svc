package core

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
)

type ProposerProvider struct {
	lastSig string
	params  *local.Params
}

func NewProposerProvider(cfg config.Config) *ProposerProvider {
	return &ProposerProvider{
		lastSig: cfg.Session().LastSignature,
		params:  local.NewParams(cfg),
	}
}

func (p *ProposerProvider) Update(sig string) {
	p.lastSig = sig
}

func (p *ProposerProvider) GetProposer(session uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(p.lastSig)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, session)
	hash := crypto.Keccak256(sigBytes, idBytes)
	return *p.params.Parties()[int(hash[len(hash)-1])%p.params.N()]
}
