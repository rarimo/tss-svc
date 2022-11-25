package core

import (
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
)

type Proposer struct {
	lastSignature string
	params        *ParamsSnapshot
}

func NewProposer(cfg config.Config) *Proposer {
	return &Proposer{
		lastSignature: cfg.Session().LastSignature,
	}
}

func (p *Proposer) NextProposer(sessionId uint64) rarimo.Party {
	sigBytes := hexutil.MustDecode(p.lastSignature)
	idBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(idBytes, sessionId)
	hash := eth.Keccak256(sigBytes, idBytes)
	return *p.params.Parties()[int(hash[len(hash)-1])%p.params.N()]
}

func (p *Proposer) WithSignature(signature string) *Proposer {
	p.lastSignature = signature
	return p
}

func (p *Proposer) WithParams(params *ParamsSnapshot) *Proposer {
	p.params = params
	return p
}
