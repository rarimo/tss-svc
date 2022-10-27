package config

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
)

type PrivateInfo struct {
	PrivateKey *ecdsa.PrivateKey
}

func (c *config) Private() *PrivateInfo {
	return c.tendermint.Do(func() interface{} {
		var config struct {
			PrivateKeyHex string `fig:"prv_key_hex"`
		}

		if err := figure.Out(&config).From(kv.MustGetStringMap(c.getter, "prv")).Please(); err != nil {
			panic(err)
		}

		prv, err := crypto.ToECDSA(hexutil.MustDecode(config.PrivateKeyHex))
		if err != nil {
			panic(err)
		}

		return &PrivateInfo{PrivateKey: prv}
	}).(*PrivateInfo)
}
