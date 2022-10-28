package config

import (
	"crypto/ecdsa"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
)

const AccountPrefix = "rarimo"

type PrivateInfo struct {
	PrivateKey    *ecdsa.PrivateKey
	AccountPrvKey cryptotypes.PrivKey
	Account       string
	ChainId       string
}

func (c *config) Private() *PrivateInfo {
	return c.tendermint.Do(func() interface{} {
		var config struct {
			PrivateKeyHex    string `fig:"prv_key_hex"`
			AccountPrvKeyHex string `fig:"account_prv_hex"`
			ChainId          string `fig:"chain_id"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(c.getter, "prv")).Please(); err != nil {
			panic(err)
		}

		prv, err := crypto.ToECDSA(hexutil.MustDecode(config.PrivateKeyHex))
		if err != nil {
			panic(err)
		}

		account := &secp256k1.PrivKey{Key: hexutil.MustDecode(config.AccountPrvKeyHex)}

		address, err := bech32.ConvertAndEncode(AccountPrefix, account.PubKey().Address().Bytes())
		if err != nil {
			panic(err)
		}

		return &PrivateInfo{
			PrivateKey:    prv,
			AccountPrvKey: account,
			Account:       address,
			ChainId:       config.ChainId,
		}
	}).(*PrivateInfo)
}
