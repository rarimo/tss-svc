package core

import (
	"bytes"
	goerr "errors"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	rarimo "github.com/rarimo/rarimo-core/x/rarimocore/types"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/logan/v3"
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
		r.log.WithError(err).Debug("Failed to recover signature public key")
		return nil, ErrInvalidSignature
	}

	// Signature is in 65 bytes format [R|S|V]. VerifySignature accepts [R|S]
	if !crypto.VerifySignature(pub, hash, signature[:64]) {
		r.log.Debug("Failed to verify signature for recovered public key")
		return nil, ErrInvalidSignature
	}

	// TODO optimize: make log(n)
	for _, p := range r.parties {
		if bytes.Equal(hexutil.MustDecode(p.PubKey), pub[1:]) {
			return p, nil
		}
	}

	return nil, ErrSignerNotAParty
}
