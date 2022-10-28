package local

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

// Secret implements singleton pattern
var secret *Secret

// Secret handles tss party private information
// and called up to be the service for signing and private key storing
type Secret struct {
	prv     *ecdsa.PrivateKey
	account cryptotypes.PrivKey
}

func NewSecret(cfg config.Config) *Secret {
	if secret == nil {
		secret = &Secret{
			prv:     cfg.Private().PrivateKey,
			account: cfg.Private().AccountPrvKey,
		}
	}
	return secret
}

func (s *Secret) ECDSAPubKey() ecdsa.PublicKey {
	return s.prv.PublicKey
}

func (s *Secret) AccountPubKey() cryptotypes.Address {
	return s.account.PubKey().Address()
}

func (s *Secret) ECDSAPubKeyStr() string {
	pub := elliptic.Marshal(secp256k1.S256(), s.prv.X, s.prv.Y)
	return hexutil.Encode(pub)
}

func (s *Secret) AccountPubKeyStr() string {
	address, _ := bech32.ConvertAndEncode(config.AccountPrefix, s.account.PubKey().Address().Bytes())
	return address
}

func (s *Secret) ECDSAPubKeyBytes() []byte {
	return elliptic.Marshal(secp256k1.S256(), s.prv.X, s.prv.Y)
}

func (s *Secret) AccountPubKeyBytes() []byte {
	return s.account.PubKey().Address().Bytes()
}

func (s *Secret) ECDSAPrvKey() *ecdsa.PrivateKey {
	return s.prv
}

func (s *Secret) AccountPrvKey() cryptotypes.PrivKey {
	return s.account
}

func (s *Secret) SignRequest(request *types.MsgSubmitRequest) error {
	hash := crypto.Keccak256(request.Details.Value)
	signature, err := crypto.Sign(hash, s.prv)
	if err != nil {
		return err
	}
	request.Signature = hexutil.Encode(signature)
	return nil
}
