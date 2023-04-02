package main

import (
	"fmt"
	"os"

	challenger "github.com/refcell/op-challenger/challenger"
	flags "github.com/refcell/op-challenger/flags"

	log "github.com/ethereum/go-ethereum/log"
	cli "github.com/urfave/cli"

	oplog "github.com/ethereum-optimism/optimism/op-service/log"
)

func main() {
	oplog.SetupDefaults()

	app := cli.NewApp()
	app.Flags = flags.Flags
	app.Version = fmt.Sprintf("1.0.0")
	app.Name = "op-challenger"
	app.Usage = "Multi-mode Challenger Agent"
	app.Description = "A multi-mode op-stack challenge agent for output dispute games written in golang."

	app.Action = func(ctx *cli.Context) error {
		return challenger.Main(ctx)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}
