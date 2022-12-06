package core

import (
	goerr "errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

var (
	ErrSignerNotAParty  = goerr.New("signer not a party")
	ErrInvalidSignature = goerr.New("invalid signature")
)

// RequestAuthorizer is responsible for authorizing requests using defined InputSet parties
type RequestAuthorizer struct {
	log *logan.Entry
	set *InputSet
}

func NewRequestAuthorizer(set *InputSet, log *logan.Entry) *RequestAuthorizer {
	return &RequestAuthorizer{
		set: set,
		log: log,
	}
}

func (r *RequestAuthorizer) Auth(request *types.MsgSubmitRequest) (rarimo.Party, error) {
	hash := crypto.Keccak256(request.Details.Value)

	signature, err := hexutil.Decode(request.Signature)
	if err != nil {
		r.log.WithError(err).Debug("failed to decode signature")
		return rarimo.Party{}, ErrInvalidSignature
	}

	pub, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		r.log.WithError(err).Debug("failed to recover signature pub key")
		return rarimo.Party{}, ErrInvalidSignature
	}

	party, ok := r.set.PartyByKey(hexutil.Encode(pub))
	if !ok {
		return rarimo.Party{}, ErrSignerNotAParty
	}

	return party, nil
}
