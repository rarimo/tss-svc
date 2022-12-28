package secret

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"sync"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	vault "github.com/hashicorp/vault/api"
	"gitlab.com/distributed_lab/logan/v3/errors"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
)

const (
	dataKey    = "data"
	preKey     = "pre"
	accountKey = "account"
	trialKey   = "trial"
)

// VaultStorage implements singleton pattern
var vaultStorage *VaultStorage

type VaultStorage struct {
	mu       sync.Mutex
	secret   *TssSecret
	kvSecret *vault.KVSecret
	client   *vault.KVv2
	path     string
}

func NewVaultStorage(cfg config.Config) *VaultStorage {
	if vaultStorage == nil {
		vaultStorage = &VaultStorage{
			client: cfg.Vault(),
			path:   os.Getenv(config.VaultSecretPath),
		}
	}

	return vaultStorage
}

// Implements Storage interface
var _ Storage = &VaultStorage{}

func (v *VaultStorage) GetTssSecret() *TssSecret {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.secret == nil {
		secret, err := v.loadSecret()
		if err != nil {
			panic(err)
		}
		v.secret = secret
	}

	return v.secret
}

func (v *VaultStorage) SetTssSecret(secret *TssSecret) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.secret = secret

	dataJson, err := json.Marshal(secret.data)
	if err != nil {
		return err
	}

	preJson, err := json.Marshal(secret.params)
	if err != nil {
		return err
	}

	v.kvSecret.Data[dataKey] = dataJson
	v.kvSecret.Data[preKey] = preJson
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
	_ = json.Unmarshal([]byte(v.kvSecret.Data[dataKey].(string)), data)

	pre := new(keygen.LocalPreParams)
	if err := json.Unmarshal([]byte(v.kvSecret.Data[preKey].(string)), data); err != nil {
		pre = loadParams()
	}

	account := &secp256k1.PrivKey{Key: hexutil.MustDecode(v.kvSecret.Data[accountKey].(string))}

	// Can be empty if TSS data set
	var prv *ecdsa.PrivateKey
	if prvBytes, err := hexutil.Decode(v.kvSecret.Data[trialKey].(string)); err != nil {
		prv, _ = crypto.ToECDSA(prvBytes)
	}

	return NewTssSecret(prv, account, data, pre), nil
}
