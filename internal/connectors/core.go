package connectors

import (
	"context"
	"fmt"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	client "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/distributed_lab/logan/v3/errors"
	ethermint "gitlab.com/rarimo/rarimo-core/ethermint/types"
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"google.golang.org/grpc"
)

const (
	successTxCode = 0
)

// CoreConnector submits signed confirmations to the rarimo core
type CoreConnector struct {
	txclient client.ServiceClient
	auth     authtypes.QueryClient
	txConfig sdkclient.TxConfig
	secret   *secret.TssSecret
	chainId  string
	coin     string
	log      *logan.Entry
}

func NewCoreConnector(cli *grpc.ClientConn, secret *secret.TssSecret, log *logan.Entry, params *config.ChainParams) *CoreConnector {
	return &CoreConnector{
		txclient: client.NewServiceClient(cli),
		auth:     authtypes.NewQueryClient(cli),
		txConfig: tx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
		secret:   secret,
		chainId:  params.ChainId,
		coin:     params.CoinName,
		log:      log,
	}
}

func (c *CoreConnector) SubmitChangeSet(set []*rarimo.Party, sig string) error {
	msg := &rarimo.MsgCreateChangePartiesOp{
		Creator:   c.secret.AccountAddress(),
		NewSet:    set,
		Signature: sig,
	}

	return c.Submit(msg)
}

func (c *CoreConnector) SubmitConfirmation(indexes []string, root string, signature string) error {
	msg := &rarimo.MsgCreateConfirmation{
		Creator:        c.secret.AccountAddress(),
		Root:           root,
		Indexes:        indexes,
		SignatureECDSA: signature,
	}

	return c.Submit(msg)
}

func (c *CoreConnector) SubmitReport(sessionId uint64, typ rarimo.ViolationType, offender string, message string) error {
	c.log.Info("Submitting violation report", logan.F{
		"violation_type": typ,
		"offender":       offender,
	})

	msg := &rarimo.MsgCreateViolationReport{
		Creator:       c.secret.AccountAddress(),
		SessionId:     fmt.Sprint(sessionId),
		ViolationType: typ,
		Offender:      offender,
		Msg:           message,
	}
	return c.Submit(msg)
}

func (c *CoreConnector) Submit(msgs ...sdk.Msg) error {
	tx, err := c.build(0, 0, msgs...)
	if err != nil {
		return err
	}

	gasUsed, _, err := c.simulate(tx)
	if err != nil {
		return err
	}

	gasLimit := ApproximateGasLimit(gasUsed)
	feeAmount := GetFeeAmount(gasLimit)

	tx, err = c.build(gasLimit, feeAmount, msgs...)
	if err != nil {
		return err
	}

	return c.submit(tx)
}

func (c *CoreConnector) submit(tx []byte) error {
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

	c.log.Debugf("Submitted transaction to the core: %s", grpcRes.TxResponse.TxHash)

	if grpcRes.TxResponse.Code != successTxCode {
		c.log.Debug(grpcRes.String())
		return errors.New(fmt.Sprintf("Got error code: %d, info: %s", grpcRes.TxResponse.Code, grpcRes.TxResponse.Info))
	}

	return nil
}

func (c *CoreConnector) simulate(tx []byte) (gasUsed uint64, gasWanted uint64, err error) {
	simResp, err := c.txclient.Simulate(context.TODO(),
		&client.SimulateRequest{
			Tx:      nil,
			TxBytes: tx,
		},
	)

	if err != nil {
		return 0, 0, err
	}

	return simResp.GasInfo.GasUsed, simResp.GasInfo.GasWanted, err
}

func (c *CoreConnector) build(gasLimit, feeAmount uint64, msgs ...sdk.Msg) ([]byte, error) {
	builder := c.txConfig.NewTxBuilder()
	err := builder.SetMsgs(msgs...)
	if err != nil {
		return nil, err
	}

	accountResp, err := c.auth.Account(context.TODO(), &authtypes.QueryAccountRequest{Address: c.secret.AccountAddress()})
	if err != nil {
		panic(err)
	}

	account := ethermint.EthAccount{}
	err = account.Unmarshal(accountResp.Account.Value)
	if err != nil {
		panic(err)
	}

	builder.SetGasLimit(gasLimit)
	builder.SetFeeAmount(sdk.Coins{sdk.NewInt64Coin(c.coin, int64(feeAmount))})

	err = builder.SetSignatures(signing.SignatureV2{
		PubKey: c.secret.AccountPubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  c.txConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: account.Sequence,
	})
	if err != nil {
		return nil, err
	}

	signerData := xauthsigning.SignerData{
		ChainID:       c.chainId,
		AccountNumber: account.AccountNumber,
		Sequence:      account.Sequence,
	}

	sigV2, err := c.secret.SignTransaction(c.txConfig, signerData, builder, account.BaseAccount)
	if err != nil {
		return nil, err
	}

	err = builder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	return c.txConfig.TxEncoder()(builder.GetTx())
}
