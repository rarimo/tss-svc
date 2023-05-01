package cli

import (
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/alecthomas/kingpin"
	tsskg "github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarimo/tss/tss-svc/internal/config"
	"gitlab.com/rarimo/tss/tss-svc/internal/core"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/empty"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/keygen"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/reshare"
	"gitlab.com/rarimo/tss/tss-svc/internal/core/sign"
	"gitlab.com/rarimo/tss/tss-svc/internal/grpc"
	"gitlab.com/rarimo/tss/tss-svc/internal/pool"
	"gitlab.com/rarimo/tss/tss-svc/internal/timer"
	"gitlab.com/rarimo/tss/tss-svc/pkg/types"
)

func Run(args []string) bool {
	defer func() {
		if rvr := recover(); rvr != nil {
			logan.New().WithRecover(rvr).Error("app panicked")
		}
	}()

	cfg := config.New(kv.MustFromEnv())
	log := cfg.Log()

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
		log.WithError(err).Error("failed to parse arguments")
		return false
	}

	core.Initialize(cfg)
	ctx := core.DefaultGlobalContext()

	switch cmd {
	case serviceCmd.FullCommand():
		go timer.NewBlockSubscriber(ctx.Timer(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewFeeManagementOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
		go pool.NewContractUpgradeOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run()
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
		err = MigrateUp(cfg)
	case migrateDownCmd.FullCommand():
		err = MigrateDown(cfg)
	default:
		log.Errorf("unknown command %s", cmd)
		return false
	}

	if err != nil {
		log.WithError(err).Error("failed to exec cmd")
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

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}
