package config

import (
	"reflect"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cast"
	"gitlab.com/distributed_lab/figure"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3/errors"
)

type PrivateInfo struct {
	AccountPrvKey cryptotypes.PrivKey `fig:"account_prv_hex"`
	ChainId       string              `fig:"chain_id"`
}

func (c *config) Private() *PrivateInfo {
	return c.private.Do(func() interface{} {
		var config PrivateInfo
		if err := figure.
			Out(&config).
			With(figure.BaseHooks, hooks).
			From(kv.MustGetStringMap(c.getter, "prv")).
			Please(); err != nil {
			panic(err)
		}
		return &config
	}).(*PrivateInfo)
}

var hooks = figure.Hooks{
	"*ecdsa.PrivateKey": func(raw interface{}) (reflect.Value, error) {
		v, err := cast.ToStringE(raw)
		if err != nil {
			return reflect.Value{}, errors.Wrap(err, "expected string")
		}

		if v == "" {
			return reflect.ValueOf(nil), nil
		}

		prv, err := crypto.ToECDSA(hexutil.MustDecode(v))
		return reflect.ValueOf(prv), err
	},
	"types.PrivKey": func(raw interface{}) (reflect.Value, error) {
		v, err := cast.ToStringE(raw)
		if err != nil {
			return reflect.Value{}, errors.Wrap(err, "expected string")
		}

		if v == "" {
			return reflect.ValueOf(nil), nil
		}

		return reflect.ValueOf(&secp256k1.PrivKey{Key: hexutil.MustDecode(v)}), nil
	},
}
