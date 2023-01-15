package main

import (
	"os"
	"runtime"

	"gitlab.com/rarimo/tss/tss-svc/internal/cli"
)

func main() {
	runtime.GOMAXPROCS(2)

	if !cli.Run(os.Args) {
		os.Exit(1)
	}
}
