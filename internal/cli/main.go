package cli

import (
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/alecthomas/kingpin"
	tsskg "github.com/bnb-chain/tss-lib/v2/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rarimo/tss-svc/internal/config"
	"github.com/rarimo/tss-svc/internal/core"
	"github.com/rarimo/tss-svc/internal/core/empty"
	"github.com/rarimo/tss-svc/internal/core/keygen"
	"github.com/rarimo/tss-svc/internal/core/reshare"
	"github.com/rarimo/tss-svc/internal/core/sign"
	"github.com/rarimo/tss-svc/internal/grpc"
	"github.com/rarimo/tss-svc/internal/pool"
	"github.com/rarimo/tss-svc/internal/timer"
	"github.com/rarimo/tss-svc/pkg/types"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3"
)

func Run(args []string) bool {
	defer func() {
		if rvr := recover(); rvr != nil {
			logan.New().WithRecover(rvr).Error("app panicked")
		}
	}()

	app := kingpin.New("tss-svc", "")
	runCmd := app.Command("run", "run command")

	// Running full service
	serviceCmd := runCmd.Command("service", "run service")

	// Running service in keygen mode
	keygenCmd := runCmd.Command("keygen", "run keygen")

	// Running pramas generation
	paramgenCmd := runCmd.Command("paramgen", "run paramgen")

	// Running ECDSA key-pair generation
	prvgenCmd := runCmd.Command("prvgen", "run prvgen")

	// Running migrations
	migrateCmd := app.Command("migrate", "migrate command")
	migrateUpCmd := migrateCmd.Command("up", "migrate db up")
	migrateDownCmd := migrateCmd.Command("down", "migrate db down")

	cmd, err := app.Parse(args[1:])
	if err != nil {
		logan.New().WithError(err).Error("failed to parse arguments")
		return false
	}

	switch cmd {
	case serviceCmd.FullCommand():
		go profiling()

		cfg := config.New(kv.MustFromEnv())
		core.Initialize(cfg)

		ctx := core.DefaultGlobalContext()

		go timer.NewBlockSubscriber(ctx.Timer(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewFeeManagementOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewContractUpgradeOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewIdentityTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewIdentityGISTTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewIdentityStateTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewIdentityAggregatedTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewWorldCoinIdentityTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewOperationCatchupper(ctx.Pool(), ctx.Client(), ctx.Log()).Run()

		manager := core.NewSessionManager()
		manager.AddSession(types.SessionType_ReshareSession, empty.NewEmptySession(ctx, cfg.Session(), types.SessionType_ReshareSession, reshare.NewSession))
		manager.AddSession(types.SessionType_DefaultSession, empty.NewEmptySession(ctx, cfg.Session(), types.SessionType_DefaultSession, sign.NewSession))

		ctx.Timer().SubscribeToBlocks("session-manager", manager.NewBlock)

		server := grpc.NewServer(ctx, manager)
		go func() {
			if err := server.RunGateway(); err != nil {
				panic(err)
			}
		}()

		err = server.RunGRPC()
	case keygenCmd.FullCommand():
		cfg := config.New(kv.MustFromEnv())
		core.Initialize(cfg)

		ctx := core.DefaultGlobalContext()

		go timer.NewBlockSubscriber(ctx.Timer(), ctx.Tendermint(), ctx.Log()).Run()

		manager := core.NewSessionManager()
		manager.AddSession(types.SessionType_KeygenSession, empty.NewEmptySession(ctx, cfg.Session(), types.SessionType_KeygenSession, keygen.NewSession))

		ctx.Timer().SubscribeToBlocks("session-manager", manager.NewBlock)

		server := grpc.NewServer(ctx, manager)
		go func() {
			if err := server.RunGateway(); err != nil {
				panic(err)
			}
		}()

		err = server.RunGRPC()
	case paramgenCmd.FullCommand():
		params, err := tsskg.GeneratePreParams(10 * time.Minute)
		if err != nil {
			panic(err)
		}

		if !params.ValidateWithProof() {
			panic("validation failed")
		}

		data, err := json.Marshal(params)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(data))
	case prvgenCmd.FullCommand():
		keypair, _ := crypto.GenerateKey()
		fmt.Println("Pub: " + hexutil.Encode(elliptic.Marshal(secp256k1.S256(), keypair.X, keypair.Y)))
		fmt.Println("Prv: " + hexutil.Encode(keypair.D.Bytes()))
	case migrateUpCmd.FullCommand():
		cfg := config.New(kv.MustFromEnv())
		err = MigrateUp(cfg)
	case migrateDownCmd.FullCommand():
		cfg := config.New(kv.MustFromEnv())
		err = MigrateDown(cfg)
	default:
		logan.New().Errorf("unknown command %s", cmd)
		return false
	}

	if err != nil {
		logan.New().WithError(err).Error("failed to exec cmd")
		return false
	}
	return true
}

func profiling() {
	r := http.NewServeMux()
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.Handle("/metrics", promhttp.Handler())

	if err := http.ListenAndServe(":8080", r); err != nil {
		panic(err)
	}
}
