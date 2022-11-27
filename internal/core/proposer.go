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
	set           *InputSet
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
	return *p.set.Parties[int(hash[len(hash)-1])%p.set.N]
}

func (p *Proposer) WithSignature(signature string) *Proposer {
	p.lastSignature = signature
	return p
}

func (p *Proposer) WithInputSet(set *InputSet) *Proposer {
	p.set = set
	return p
}
