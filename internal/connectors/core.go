package connectors

import (
	"context"
	"fmt"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
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
	rarimo "gitlab.com/rarimo/rarimo-core/x/rarimocore/types"
	"gitlab.com/rarimo/tss/tss-svc/internal/secret"
	"google.golang.org/grpc"
)

const (
	chainId       = "rarimo"
	coinName      = "stake"
	successTxCode = 0
	minGasPrice   = 1
	gasLimit      = 100_000_000
)

// CoreConnector submits signed confirmations to the rarimo core
type CoreConnector struct {
	txclient client.ServiceClient
	auth     authtypes.QueryClient
	txConfig sdkclient.TxConfig
	secret   *secret.TssSecret
	log      *logan.Entry
}

func NewCoreConnector(cli *grpc.ClientConn, secret *secret.TssSecret, log *logan.Entry) *CoreConnector {
	return &CoreConnector{
		txclient: client.NewServiceClient(cli),
		auth:     authtypes.NewQueryClient(cli),
		txConfig: tx.NewTxConfig(codec.NewProtoCodec(codectypes.NewInterfaceRegistry()), []signing.SignMode{signing.SignMode_SIGN_MODE_DIRECT}),
		secret:   secret,
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
	builder := c.txConfig.NewTxBuilder()
	err := builder.SetMsgs(msgs...)
	if err != nil {
		return err
	}

	builder.SetGasLimit(gasLimit)
	builder.SetFeeAmount(types.Coins{types.NewInt64Coin(coinName, int64(gasLimit*minGasPrice))})

	accountResp, err := c.auth.Account(context.TODO(), &authtypes.QueryAccountRequest{Address: c.secret.AccountAddress()})
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
		ChainID:       chainId,
		AccountNumber: account.AccountNumber,
		Sequence:      account.Sequence,
	}

	sigV2, err := c.secret.SignTransaction(c.txConfig, signerData, builder, &account)
	if err != nil {
		return err
	}

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

	c.log.Debugf("Submitted transaction to the core: %s", grpcRes.TxResponse.TxHash)

	if grpcRes.TxResponse.Code != successTxCode {
		c.log.Debug(grpcRes.String())
		return errors.New(fmt.Sprintf("Got error code: %d, info: %s", grpcRes.TxResponse.Code, grpcRes.TxResponse.Info))
	}

	return nil
}
