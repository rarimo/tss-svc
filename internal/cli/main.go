package cli

import (
	"crypto/elliptic"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/pprof"
	"runtime"
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
	go profiling()

	log := logan.New()

	defer func() {
		if rvr := recover(); rvr != nil {
			log.WithRecover(rvr).Error("app panicked")
		}
	}()

	cfg := config.New(kv.MustFromEnv())

	log = cfg.Log()

	app := kingpin.New("tss-svc", "")

	runCmd := app.Command("run", "run command")
	serviceCmd := runCmd.Command("service", "run service")
	keygenCmd := runCmd.Command("keygen", "run keygen")
	paramgenCmd := runCmd.Command("paramgen", "run paramgen")
	prvgenCmd := runCmd.Command("prvgen", "run prvgen")

	migrateCmd := app.Command("migrate", "migrate command")
	migrateUpCmd := migrateCmd.Command("up", "migrate db up")
	migrateDownCmd := migrateCmd.Command("down", "migrate db down")

	cmd, err := app.Parse(args[1:])
	if err != nil {
		log.WithError(err).Error("failed to parse arguments")
		return false
	}

	log.Infof("GOMAXPROCS = %d", runtime.GOMAXPROCS(0))

	switch cmd {
	case serviceCmd.FullCommand():
		go timer.NewBlockSubscriber(cfg).Run()            // Timer initialized
		go pool.NewTransferOperationSubscriber(cfg).Run() // Pool initialized
		go pool.NewOperationCatchupper(cfg).Run()

		manager := core.NewSessionManager()
		manager.AddSession(types.SessionType_ReshareSession, empty.NewEmptySession(cfg, types.SessionType_ReshareSession, reshare.NewSession))
		manager.AddSession(types.SessionType_DefaultSession, empty.NewEmptySession(cfg, types.SessionType_DefaultSession, sign.NewSession))

		timer.GetTimer().SubscribeToBlocks("session-manager", manager.NewBlock)

		err = grpc.NewServer(manager, cfg).Run()
	case keygenCmd.FullCommand():
		go timer.NewBlockSubscriber(cfg).Run()            // Timer initialized
		go pool.NewTransferOperationSubscriber(cfg).Run() // Pool initialized
		go pool.NewOperationCatchupper(cfg).Run()

		manager := core.NewSessionManager()
		manager.AddSession(types.SessionType_KeygenSession, empty.NewEmptySession(cfg, types.SessionType_KeygenSession, keygen.NewSession))

		timer.GetTimer().SubscribeToBlocks("session-manager", manager.NewBlock)

		err = grpc.NewServer(manager, cfg).Run()
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
