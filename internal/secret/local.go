package secret

import (
	"crypto/ecdsa"
	"encoding/json"
	goerr "errors"
	"os"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
)

const (
	AccountPrefix   = "rarimo"
	PartyTSSDataENV = "PARTY_TSS_DATA_PATH"
	PreParamsENV    = "LOCAL_PRE_PARAMS_PATH"
)

var ErrNoTssDataPath = goerr.New("tss data path is empty")

// LocalStorage implements singleton pattern
var localStorage *LocalStorage

type LocalStorage struct {
	mu          sync.Mutex
	account     cryptotypes.PrivKey
	TrialPrvKey *ecdsa.PrivateKey
	secret      *TssSecret
}

func NewLocalStorage(cfg config.Config) *LocalStorage {
	if localStorage == nil {
		data, err := loadData()
		if err == ErrNoTssDataPath {
			panic(ErrNoTssDataPath)
		}

		var prv *ecdsa.PrivateKey
		if data == nil || data.Xi == nil {
			prv = cfg.Private().PrivateKey
		}

		localStorage = &LocalStorage{
			account:     cfg.Private().AccountPrvKey,
			TrialPrvKey: cfg.Private().PrivateKey,
			secret:      NewTssSecret(prv, cfg.Private().AccountPrvKey, data, loadParams()),
		}
	}

	return localStorage
}

// Implements Storage interface
var _ Storage = &LocalStorage{}

func (l *LocalStorage) GetTssSecret() *TssSecret {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.secret
}

func (l *LocalStorage) SetTssSecret(secret *TssSecret) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.secret = secret
	return saveData(secret.data)
}

func loadParams() *keygen.LocalPreParams {
	if params := openParams(); params != nil {
		return params
	}

	params, err := keygen.GeneratePreParams(10 * time.Minute)
	if err != nil {
		panic(err)
	}

	return params
}

func openParams() *keygen.LocalPreParams {
	path := os.Getenv(PreParamsENV)
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	res := new(keygen.LocalPreParams)
	if err = json.Unmarshal(data, res); err != nil {
		return nil
	}

	return res
}

func loadData() (*keygen.LocalPartySaveData, error) {
	path := os.Getenv(PartyTSSDataENV)
	if path == "" {
		return nil, ErrNoTssDataPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	res := new(keygen.LocalPartySaveData)
	return res, json.Unmarshal(data, res)
}

func saveData(data *keygen.LocalPartySaveData) error {
	path := os.Getenv(PartyTSSDataENV)
	if path == "" {
		return ErrNoTssDataPath
	}

	buf, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return os.WriteFile(path, buf, os.ModeAppend)
}
