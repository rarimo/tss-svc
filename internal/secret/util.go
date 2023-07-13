package secret

import (
	"encoding/json"
	goerr "errors"
	"os"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
)

var ErrNoTssDataPath = goerr.New("tss data path is empty")

const (
	AccountPrefix   = "rarimo"
	PartyTSSDataENV = "PARTY_TSS_DATA_PATH"
	PreParamsENV    = "LOCAL_PRE_PARAMS_PATH"
)

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

// DEPRECATED: use Vault configuration
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

// DEPRECATED: use Vault configuration
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

// DEPRECATED: use Vault configuration
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
