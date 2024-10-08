package cli

import (
	"context"
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"syscall"
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

func Run(args []string) {
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
		logan.New().WithError(err).Fatal("failed to parse arguments")
	}

	c, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sigint := make(chan os.Signal, 1)

		// interrupt signal sent from terminal
		signal.Notify(sigint, os.Interrupt)
		// sigterm signal sent from kubernetes
		signal.Notify(sigint, syscall.SIGTERM)

		<-sigint

		cancel()
		os.Exit(0)
	}()

	switch cmd {
	case serviceCmd.FullCommand():
		go profiling(c)

		cfg := config.New(kv.MustFromEnv())
		core.Initialize(cfg)

		ctx := core.DefaultGlobalContext(c)
		go timer.NewBlockSubscriber(ctx.Timer(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewFeeManagementOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewIdentityGISTTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewIdentityStateTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewIdentityAggregatedTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewWorldCoinIdentityTransferOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewPassportRootUpdateOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewCSCARootUpdateOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewArbitraryOperationSubscriber(ctx.Pool(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())
		go pool.NewOperationCatchupper(ctx.Pool(), ctx.Client(), ctx.Log()).Run(ctx.Context())

		manager := core.NewSessionManager()
		manager.AddSession(types.SessionType_ReshareSession, empty.NewEmptySession(ctx, cfg.Session(), types.SessionType_ReshareSession, reshare.NewSession))
		manager.AddSession(types.SessionType_DefaultSession, empty.NewEmptySession(ctx, cfg.Session(), types.SessionType_DefaultSession, sign.NewSession))

		ctx.Timer().SubscribeToBlocks("session-manager", manager.NewBlock)

		server := grpc.NewServer(ctx.Log(), ctx.Listener(), ctx.PG(), ctx.SecretStorage(), ctx.Pool(), ctx.Swagger(), manager)
		go func() {
			if err := server.RunGateway(ctx.Context()); err != nil {
				ctx.Log().WithError(err).Fatal("rest gateway server error")
			}
		}()

		err = server.RunGRPC(ctx.Context())
	case keygenCmd.FullCommand():
		go profiling(c)

		cfg := config.New(kv.MustFromEnv())
		core.Initialize(cfg)

		ctx := core.DefaultGlobalContext(c)

		go timer.NewBlockSubscriber(ctx.Timer(), ctx.Tendermint(), ctx.Log()).Run(ctx.Context())

		manager := core.NewSessionManager()
		manager.AddSession(types.SessionType_KeygenSession, empty.NewEmptySession(ctx, cfg.Session(), types.SessionType_KeygenSession, keygen.NewSession))

		ctx.Timer().SubscribeToBlocks("session-manager", manager.NewBlock)

		server := grpc.NewServer(ctx.Log(), ctx.Listener(), ctx.PG(), ctx.SecretStorage(), ctx.Pool(), ctx.Swagger(), manager)
		go func() {
			if err := server.RunGateway(ctx.Context()); err != nil {
				ctx.Log().WithError(err).Fatal("rest gateway server error")
			}
		}()

		err = server.RunGRPC(ctx.Context())
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
		logan.New().Fatalf("unknown command %s", cmd)
	}

	if err != nil {
		logan.New().WithError(err).Error("failed to exec cmd")
	}
}

func profiling(ctx context.Context) {
	r := http.NewServeMux()
	r.HandleFunc("/debug/pprof/", pprof.Index)
	r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/debug/pprof/profile", pprof.Profile)
	r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	r.Handle("/metrics", promhttp.Handler())

	srv := &http.Server{Addr: ":8080", Handler: r}
	if err := srv.ListenAndServe(); err != nil {
		logan.New().WithError(err).Error("profiler server error")
	}

	if err := srv.Shutdown(ctx); err != nil {
		logan.New().WithError(err).Error("profiler server shutdown error")
	}
}
