package connectors

import (
	"context"
	"fmt"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	client "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	rarimo "gitlab.com/rarify-protocol/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/local"
)

const (
	coinName      = "stake"
	successTxCode = 0
	minGasPrice   = 1
	gasLimit      = 100_000_000
)

// CoreConnector submits signed confirmations to the rarimo core
type CoreConnector struct {
	params   *local.Params
	secret   *local.Secret
	txclient client.ServiceClient
	auth     authtypes.QueryClient
	txConfig sdkclient.TxConfig
	log      *logan.Entry
}

func NewCoreConnector(cfg config.Config) *CoreConnector {
	return &CoreConnector{
		params:   local.NewParams(cfg),
		secret:   local.NewSecret(cfg),
		txclient: client.NewServiceClient(cfg.Cosmos()),
		auth:     authtypes.NewQueryClient(cfg.Cosmos()),
		txConfig: tx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
		log:      cfg.Log(),
	}
}

func (c *CoreConnector) SubmitChangeSet(new []*rarimo.Party) error {
	msg := &rarimo.MsgCreateChangePartiesOp{
		Creator: c.secret.AccountAddressStr(),
		NewSet:  new,
	}

	return c.Submit(msg)
}

func (c *CoreConnector) SubmitConfirmation(indexes []string, root string, signature string, meta *rarimo.ConfirmationMeta) error {
	msg := &rarimo.MsgCreateConfirmation{
		Creator:        c.secret.AccountAddressStr(),
		Root:           root,
		Indexes:        indexes,
		SignatureECDSA: signature,
		Meta:           meta,
	}

	return c.Submit(msg)
}

func (c *CoreConnector) Submit(msgs ...sdk.Msg) error {
	builder := c.txConfig.NewTxBuilder()
	err := builder.SetMsgs(msgs...)
	if err != nil {
		return err
	}

	builder.SetGasLimit(gasLimit)
	builder.SetFeeAmount(types.Coins{types.NewInt64Coin(coinName, int64(gasLimit*minGasPrice))})

	accountResp, err := c.auth.Account(context.TODO(), &authtypes.QueryAccountRequest{Address: c.secret.AccountAddressStr()})
	if err != nil {
		panic(err)
	}

	account := authtypes.BaseAccount{}
	err = account.Unmarshal(accountResp.Account.Value)
	if err != nil {
		panic(err)
	}

	err = builder.SetSignatures(signing.SignatureV2{
		PubKey: c.secret.AccountPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  c.txConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: account.Sequence,
	})
	if err != nil {
		return err
	}

	signerData := xauthsigning.SignerData{
		ChainID:       c.params.ChainId(),
		AccountNumber: account.AccountNumber,
		Sequence:      account.Sequence,
	}

	sigV2, err := clienttx.SignWithPrivKey(
		c.txConfig.SignModeHandler().DefaultMode(), signerData,
		builder, c.secret.AccountPrvKey(), c.txConfig, account.Sequence,
	)

	err = builder.SetSignatures(sigV2)
	if err != nil {
		return err
	}

	tx, err := c.txConfig.TxEncoder()(builder.GetTx())
	if err != nil {
		return err
	}

	grpcRes, err := c.txclient.BroadcastTx(
		context.TODO(),
		&client.BroadcastTxRequest{
			Mode:    client.BroadcastMode_BROADCAST_MODE_BLOCK,
			TxBytes: tx,
		},
	)
	if err != nil {
		return err
	}

	if grpcRes.TxResponse.Code != successTxCode {
		c.log.Debug(grpcRes.String())
		return errors.New(fmt.Sprintf("Got error code: %d, info: %s", grpcRes.TxResponse.Code, grpcRes.TxResponse.Info))
	}

	return nil
}
