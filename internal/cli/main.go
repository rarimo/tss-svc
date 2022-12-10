package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/alecthomas/kingpin"
	tsskg "github.com/bnb-chain/tss-lib/ecdsa/keygen"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3"
	"gitlab.com/rarify-protocol/tss-svc/internal/config"
	"gitlab.com/rarify-protocol/tss-svc/internal/core"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/empty"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/keygen"
	"gitlab.com/rarify-protocol/tss-svc/internal/core/sign"
	"gitlab.com/rarify-protocol/tss-svc/internal/grpc"
	"gitlab.com/rarify-protocol/tss-svc/internal/pool"
	"gitlab.com/rarify-protocol/tss-svc/internal/timer"
)

func Run(args []string) bool {
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

	migrateCmd := app.Command("migrate", "migrate command")
	migrateUpCmd := migrateCmd.Command("up", "migrate db up")
	migrateDownCmd := migrateCmd.Command("down", "migrate db down")

	cmd, err := app.Parse(args[1:])
	if err != nil {
		log.WithError(err).Error("failed to parse arguments")
		return false
	}

	switch cmd {
	case serviceCmd.FullCommand():
		go timer.NewBlockSubscriber(cfg).Run()
		go pool.NewTransferOperationSubscriber(cfg).Run()
		go pool.NewOperationCatchupper(cfg).Run()

		manager := core.NewSessionManager(empty.NewEmptySession(cfg, sign.NewSession))
		timer.NewTimer(cfg).SubscribeToBlocks("session-manager", manager.NewBlock)
		err = grpc.NewServer(manager, cfg).Run()
	case keygenCmd.FullCommand():
		go timer.NewBlockSubscriber(cfg).Run()
		go pool.NewTransferOperationSubscriber(cfg).Run()
		go pool.NewOperationCatchupper(cfg).Run()

		manager := core.NewSessionManager(empty.NewEmptySession(cfg, keygen.NewSession))
		timer.NewTimer(cfg).SubscribeToBlocks("session-manager", manager.NewBlock)
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
