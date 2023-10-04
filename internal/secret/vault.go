package secret

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"sync"

	"github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	vault "github.com/hashicorp/vault/api"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
)

const (
	dataKey    = "data"
	preKey     = "pre"
	accountKey = "account"
	trialKey   = "trial"
	enableTLS  = "tls"
)

type VaultStorage struct {
	once     sync.Once
	log      *logan.Entry
	secret   *TssSecret
	kvSecret *vault.KVSecret
	client   *vault.KVv2
	path     string
}

func NewVaultStorage(cfg config.Config) *VaultStorage {
	return &VaultStorage{
		client: cfg.Vault(),
		log:    cfg.Log(),
		path:   os.Getenv(config.VaultSecretPath),
	}
}

// Implements Storage interface
var _ Storage = &VaultStorage{}

func (v *VaultStorage) GetTssSecret() *TssSecret {
	v.once.Do(func() {
		var err error
		v.secret, err = v.loadSecret()
		if err != nil {
			panic(err)
		}
	})

	return v.secret
}

func (v *VaultStorage) SetTssSecret(secret *TssSecret) error {
	v.secret = secret

	dataJson, err := json.Marshal(secret.data)
	if err != nil {
		return err
	}

	preJson, err := json.Marshal(secret.params)
	if err != nil {
		return err
	}

	v.kvSecret.Data[dataKey] = string(dataJson)
	v.kvSecret.Data[preKey] = string(preJson)
	// Account and Trial Private Key will not be changed by tss instance, so - skipped

	v.kvSecret, err = v.client.Put(context.TODO(), v.path, v.kvSecret.Data)
	return err
}

func (v *VaultStorage) loadSecret() (*TssSecret, error) {
	var err error
	v.kvSecret, err = v.client.Get(context.Background(), v.path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret data")
	}

	data := new(keygen.LocalPartySaveData)

	// Data can be empty
	if err = json.Unmarshal([]byte(v.kvSecret.Data[dataKey].(string)), data); err != nil {
		v.log.Info("[Vault] TSS Save Data is empty")
	}

	pre := new(keygen.LocalPreParams)
	if err := json.Unmarshal([]byte(v.kvSecret.Data[preKey].(string)), pre); err != nil {
		v.log.Info("[Vault] Generating tss pre-params")
		pre = loadParams()
	}

	if !pre.ValidateWithProof() {
		return nil, errors.New("pre-params in LocalPreParams validation failed. Please, re-generate pre-params.")
	}

	account := &secp256k1.PrivKey{Key: hexutil.MustDecode(v.kvSecret.Data[accountKey].(string))}

	// Can be empty if TSS data set
	var prv *ecdsa.PrivateKey
	if prvBytes, err := hexutil.Decode(v.kvSecret.Data[trialKey].(string)); err == nil {
		v.log.Info("[Vault] Trial private key found")
		prv, _ = crypto.ToECDSA(prvBytes)
	}

	tls := false

	if enableI, ok := v.kvSecret.Data[enableTLS]; ok {
		tls = enableI.(bool)
	}

	return NewTssSecret(prv, account, data, pre, tls), nil
}
