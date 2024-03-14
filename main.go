package main

import (
	"os"
	"runtime"
	"strconv"

	"github.com/rarimo/tss-svc/internal/cli"
)

const GomaxprocsEnv = "TSS_GOMAXPROCS"

func main() {
	if gomaxprocsStr := os.Getenv(GomaxprocsEnv); gomaxprocsStr != "" {
		if gomaxprocs, err := strconv.Atoi(gomaxprocsStr); err == nil {
			runtime.GOMAXPROCS(gomaxprocs)
		}
	}

	cli.Run(os.Args)
}
