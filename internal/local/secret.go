package local

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"encoding/json"
	goerr "errors"
	"math/big"
	"os"
	"sync"
	"time"

	"github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/bnb-chain/tss-lib/tss"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/pkg/types"
)

const (
	AccountPrefix   = "rarimo"
	PartyTSSDataENV = "PARTY_TSS_DATA_PATH"
	PreParamsENV    = "LOCAL_PRE_PARAMS_PATH"
	FileType        = ".json"
)

// Secret implements singleton pattern
var secret *Secret
var ErrNoTssDataPath = goerr.New("tss data path is empty")

// Secret handles tss party private information
// and called up to be the service for signing and private key storing
type Secret struct {
	*sync.Mutex
	prv     *ecdsa.PrivateKey
	account cryptotypes.PrivKey
	data    *keygen.LocalPartySaveData
	pre     *keygen.LocalPreParams
	log     *logan.Entry
}

func NewSecret(cfg config.Config) *Secret {
	if secret == nil {
		secret = &Secret{
			account: cfg.Private().AccountPrvKey,
			log:     cfg.Log(),
		}

		prv, err := crypto.ToECDSA(secret.MustGetLocalPartyData().Xi.Bytes())
		if err != nil {
			panic(err)
		}

		secret.prv = prv
	}
	return secret
}

func (s *Secret) ECDSAPubKey() ecdsa.PublicKey {
	s.Lock()
	defer s.Unlock()
	return s.prv.PublicKey
}

func (s *Secret) AccountPubKey() cryptotypes.PubKey {
	return s.account.PubKey()
}

func (s *Secret) AccountAddress() cryptotypes.Address {
	return s.account.PubKey().Address()
}

func (s *Secret) ECDSAPubKeyStr() string {
	s.Lock()
	defer s.Unlock()
	pub := elliptic.Marshal(secp256k1.S256(), s.prv.X, s.prv.Y)
	return hexutil.Encode(pub)
}

func (s *Secret) AccountAddressStr() string {
	address, _ := bech32.ConvertAndEncode(AccountPrefix, s.account.PubKey().Address().Bytes())
	return address
}

func (s *Secret) ECDSAPubKeyBytes() []byte {
	s.Lock()
	defer s.Unlock()
	return elliptic.Marshal(secp256k1.S256(), s.prv.X, s.prv.Y)
}

func (s *Secret) ECDSAPrvKey() *ecdsa.PrivateKey {
	s.Lock()
	defer s.Unlock()
	return s.prv
}

func (s *Secret) AccountPrvKey() cryptotypes.PrivKey {
	return s.account
}

func (s *Secret) PartyId() *tss.PartyID {
	return tss.NewPartyID(s.AccountAddressStr(), "", new(big.Int).SetBytes(s.account.PubKey().Address().Bytes()))
}

func (s *Secret) SignRequest(request *types.MsgSubmitRequest) error {
	s.Lock()
	defer s.Unlock()
	hash := crypto.Keccak256(request.Details.Value)
	signature, err := crypto.Sign(hash, s.prv)
	if err != nil {
		return err
	}
	request.Signature = hexutil.Encode(signature)
	return nil
}

func (s *Secret) GetLocalPartyData() (*keygen.LocalPartySaveData, error) {
	path := os.Getenv(PartyTSSDataENV)
	if path == "" {
		return nil, ErrNoTssDataPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	res := new(keygen.LocalPartySaveData)
	if err = json.Unmarshal(data, res); err != nil {
		return nil, err
	}

	return res, nil
}

func (s *Secret) MustGetLocalPartyData() *keygen.LocalPartySaveData {
	s.Lock()
	defer s.Unlock()

	if s.data == nil {
		res, err := s.GetLocalPartyData()
		if err != nil {
			panic(err)
		}

		s.data = res
	}
	return s.data
}

func (s *Secret) GetGlobalPubKey() *ecdsa.PublicKey {
	point := s.MustGetLocalPartyData().ECDSAPub
	return &ecdsa.PublicKey{
		Curve: secp256k1.S256(),
		X:     point.X(),
		Y:     point.Y(),
	}
}

func (s *Secret) MustGetLocalPartyPreParams() *keygen.LocalPreParams {
	s.Lock()
	defer s.Unlock()

	if s.pre == nil {
		params, err := s.GetLocalPartyPreParams()
		if err != nil {
			panic(err)
		}

		s.pre = params
	}
	return s.pre
}

func (s *Secret) GetLocalPartyPreParams() (*keygen.LocalPreParams, error) {
	if params := openParams(); params != nil {
		s.log.Info("Params opened from file")
		return params, nil
	}
	s.log.Info("Generating pre params")
	return keygen.GeneratePreParams(10 * time.Minute)
}

func openParams() *keygen.LocalPreParams {
	path := os.Getenv(PreParamsENV)
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	res := new(keygen.LocalPreParams)
	if err = json.Unmarshal(data, res); err != nil {
		panic(err)
	}

	return res
}

func (s *Secret) UpdateLocalPartyData(key *keygen.LocalPartySaveData) error {
	s.Lock()
	defer s.Unlock()

	path := os.Getenv(PartyTSSDataENV)
	if path == "" {
		return ErrNoTssDataPath
	}

	data, err := json.Marshal(key)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, data, os.ModeAppend); err != nil {
		return err
	}
	s.data = key
	s.prv, _ = crypto.ToECDSA(secret.MustGetLocalPartyData().Xi.Bytes())
	return nil
}
