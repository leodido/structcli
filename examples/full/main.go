//go:generate go run ./cmd/generate

package main

import (
	"log"

	"github.com/leodido/structcli"
	full_example_cli "github.com/leodido/structcli/examples/full/cli"
)

func main() {
	log.SetFlags(0)
	c, e := full_example_cli.NewRootC(true)
	if e != nil {
		log.Fatalln(e)
	}

	structcli.ExecuteOrExit(c)
}
