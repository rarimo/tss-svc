package main

import (
	"os"
	"runtime"
	"strconv"

	"gitlab.com/rarimo/tss/tss-svc/internal/cli"
)

const GomaxprocsEnv = "TSS_GOMAXPROCS"

func main() {
	if gomaxprocsStr := os.Getenv(GomaxprocsEnv); gomaxprocsStr != "" {
		if gomaxprocs, err := strconv.Atoi(gomaxprocsStr); err == nil {
			runtime.GOMAXPROCS(gomaxprocs)
		}
	}

	if !cli.Run(os.Args) {
		os.Exit(1)
	}
}
