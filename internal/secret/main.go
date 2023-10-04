package secret

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	goerr "errors"
	"math/big"

	"github.com/bnb-chain/tss-lib/v2/common"
	tsskeygen "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	tsssign "github.com/bnb-chain/tss-lib/v2/ecdsa/signing"
	"github.com/bnb-chain/tss-lib/v2/tss"
	"github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	eth "github.com/ethereum/go-ethereum/crypto"
	"github.com/rarimo/tss-svc/pkg/types"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	ErrUninitializedPrivateKey = goerr.New("private key or TSS data should be initialized")
	ErrNoTssData               = goerr.New("tss data is empty")
)

type TssSecret struct {
	tssPrv     *ecdsa.PrivateKey
	accountPrv cryptotypes.PrivKey
	data       *tsskeygen.LocalPartySaveData
	params     *tsskeygen.LocalPreParams
	tls        bool
}

func NewTssSecret(prv *ecdsa.PrivateKey, account cryptotypes.PrivKey, data *tsskeygen.LocalPartySaveData, params *tsskeygen.LocalPreParams, tls bool) *TssSecret {
	if data != nil && data.Xi != nil {
		var err error
		prv, err = eth.ToECDSA(data.Xi.Bytes())
		if err != nil {
			panic(err)
		}
	}

	if prv == nil {
		panic(ErrUninitializedPrivateKey)
	}

	return &TssSecret{
		tssPrv:     prv,
		accountPrv: account,
		data:       data,
		params:     params,
		tls:        tls,
	}
}

func (t *TssSecret) NewWithData(data *tsskeygen.LocalPartySaveData) *TssSecret {
	prv, err := eth.ToECDSA(data.Xi.Bytes())
	if err != nil {
		panic(err)
	}

	return &TssSecret{
		tssPrv:     prv,
		accountPrv: t.accountPrv,
		data:       data,
		params:     &data.LocalPreParams,
	}
}

func (t *TssSecret) Sign(request *types.MsgSubmitRequest) error {
	details, err := anypb.New(request.Data)
	if err != nil {
		return err
	}

	hash := eth.Keccak256(details.Value)

	signature, err := eth.Sign(hash, t.tssPrv)
	if err != nil {
		return err
	}
	request.Signature = hexutil.Encode(signature)
	return nil
}

func (t *TssSecret) SignTransaction(txConfig client.TxConfig, data xauthsigning.SignerData, builder client.TxBuilder, account *authtypes.BaseAccount) (signing.SignatureV2, error) {
	return clienttx.SignWithPrivKey(
		txConfig.SignModeHandler().DefaultMode(), data,
		builder, t.accountPrv, txConfig, account.Sequence,
	)
}

func (t *TssSecret) TssPubKey() string {
	// Marshalled point contains constant 0x04 first byte, we have to remove it
	marshalled := elliptic.Marshal(eth.S256(), t.tssPrv.X, t.tssPrv.Y)
	return hexutil.Encode(marshalled[1:])
}

func (t *TssSecret) AccountAddress() string {
	address, _ := bech32.ConvertAndEncode(AccountPrefix, t.accountPrv.PubKey().Address().Bytes())
	return address
}

func (t *TssSecret) AccountPubKey() cryptotypes.PubKey {
	return t.accountPrv.PubKey()
}

func (t *TssSecret) GlobalPubKey() string {
	if t.data == nil {
		return ""
	}

	// Marshalled point contains constant 0x04 first byte, we have to remove it
	marshalled := elliptic.Marshal(eth.S256(), t.data.ECDSAPub.X(), t.data.ECDSAPub.Y())
	return hexutil.Encode(marshalled[1:])
}

func (t *TssSecret) GetKeygenParty(params *tss.Parameters, out chan<- tss.Message, end chan<- *tsskeygen.LocalPartySaveData) tss.Party {
	return tsskeygen.NewLocalParty(params, out, end, *t.params)
}

func (t *TssSecret) GetSignParty(msg *big.Int, params *tss.Parameters, out chan<- tss.Message, end chan<- *common.SignatureData) tss.Party {
	return tsssign.NewLocalParty(msg, params, *t.data, out, end)
}

func (t *TssSecret) TLS() bool {
	return t.tls
}

// Storage is responsible for managing TSS secret data and Rarimo core account secret data
type Storage interface {
	GetTssSecret() *TssSecret
	SetTssSecret(secret *TssSecret) error
}
