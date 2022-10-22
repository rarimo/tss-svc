package main

import (
	"os"

	"gitlab.com/rarify-protocol/tss-svc/internal/cli"
)

func main() {
	if !cli.Run(os.Args) {
		os.Exit(1)
	}
}
