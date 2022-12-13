package main

import (
	"os"

	"gitlab.com/rarimo/tss/tss-svc/internal/cli"
)

func main() {
	if !cli.Run(os.Args) {
		os.Exit(1)
	}
}
