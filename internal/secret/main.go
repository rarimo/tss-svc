package secret

import (
	"crypto/ecdsa"
	"crypto/elliptic"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
)

type TssSecret struct {
	Prv    *ecdsa.PrivateKey
	Data   *keygen.LocalPartySaveData
	Params *keygen.LocalPreParams
	prev   *TssSecret
}

func NewTssSecret(data *keygen.LocalPartySaveData, params *keygen.LocalPreParams, prev *TssSecret) *TssSecret {
	var (
		prv *ecdsa.PrivateKey
		err error
	)

	if data != nil && data.Xi != nil {
		prv, err = eth.ToECDSA(data.Xi.Bytes())
		if err != nil {
			panic(err)
		}
	}

	return &TssSecret{
		Prv:    prv,
		Data:   data,
		Params: params,
		prev:   prev,
	}
}

func (t *TssSecret) PubKeyStr() string {
	if t.Prv == nil {
		return ""
	}

	return hexutil.Encode(elliptic.Marshal(eth.S256(), t.Prv.X, t.Prv.Y))
}

func (t *TssSecret) GlobalPubKeyStr() string {
	return hexutil.Encode(elliptic.Marshal(eth.S256(), t.Data.ECDSAPub.X(), t.Data.ECDSAPub.Y()))
}

func (t *TssSecret) Previous() *TssSecret {
	return t.prev
}

// Storage is responsible for managing TSS secret data and Rarimo core account secret data
type Storage interface {
	// Core account management
	AccountAddressStr() string
	AccountPrvKey() cryptotypes.PrivKey

	// TSS account management
	GetTssSecret() *TssSecret
	GetTrialPrivateKey() *ecdsa.PrivateKey
	SetTssSecret(secret *TssSecret) error
}
