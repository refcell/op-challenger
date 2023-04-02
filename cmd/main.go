package main

import (
	"context"
	"fmt"
	"os"
	"time"

	// flags "github.com/refcell/op-challenger/flags"
	// challenger "github.com/refcell/op-challenger/challenger"

	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	log "github.com/ethereum/go-ethereum/log"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	peer "github.com/libp2p/go-libp2p/core/peer"
	"github.com/refcell/malleable/flags"
	cli "github.com/urfave/cli"

	// flags "github.com/ethereum-optimism/optimism/op-batcher/flags"
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
		return ExecuteChallenger(ctx)
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Crit("Application failed", "message", err)
	}
}

// ExecuteChallenger creates a new Challenger from cli and runs it.
func ExecuteChallenger(cliCtx *cli.Context) error {
	cfg, err := NewConfig(cliCtx)
	if err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := cfg.log
	l.Info("Initializing Challenger...")

	var topicScoreParams *pubsub.TopicScoreParams
	callback := func(ctx context.Context, from peer.ID, payload *eth.ExecutionPayload) error {
		fmt.Printf("Received payload with block number: %d\n", payload.BlockNumber)
		return nil
	}
	malleableNode, err := NewChallenger(l, cfg.L2ChainID, topicScoreParams, cfg.PrivKey, callback)
	if err != nil {
		l.Error("Unable to create Malleable Node", "error", err)
		return err
	}

	malleableNodeID := malleableNode.ID()
	fmt.Printf("Malleable Node ID: %s\n", malleableNodeID)

	// Wait a bit...
	time.Sleep(5 * time.Second)

	return nil
}
