package core

import (
	goerr "errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/logan/v3"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	ErrSignerNotAParty  = goerr.New("signer not a party")
	ErrInvalidSignature = goerr.New("invalid signature")
)

// RequestAuthorizer is responsible for authorizing requests using defined InputSet parties
type RequestAuthorizer struct {
	log     *logan.Entry
	parties []*rarimo.Party
}

func NewRequestAuthorizer(parties []*rarimo.Party, log *logan.Entry) *RequestAuthorizer {
	return &RequestAuthorizer{
		parties: parties,
		log:     log,
	}
}

func (r *RequestAuthorizer) Auth(request *types.MsgSubmitRequest) (*rarimo.Party, error) {
	details, err := anypb.New(request.Data)
	if err != nil {
		return nil, err
	}

	hash := crypto.Keccak256(details.Value)

	signature, err := hexutil.Decode(request.Signature)
	if err != nil {
		r.log.WithError(err).Debug("Failed to decode signature")
		return nil, ErrInvalidSignature
	}

	pub, err := crypto.Ecrecover(hash, signature)
	if err != nil {
		r.log.WithError(err).Debug("Failed to recover signature pub key")
		return nil, ErrInvalidSignature
	}

	if !crypto.VerifySignature(pub, hash, signature) {
		r.log.WithError(err).Debug("Failed to verify signature for recovered public ket")
		return nil, ErrInvalidSignature
	}

	// TODO optimize: make log(n)
	key := hexutil.Encode(pub)
	for _, p := range r.parties {
		if p.PubKey == key {
			return p, nil
		}
	}

	return nil, ErrSignerNotAParty
}
