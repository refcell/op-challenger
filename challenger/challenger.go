package challenger

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	goEth "github.com/ethereum/go-ethereum"
	abi "github.com/ethereum/go-ethereum/accounts/abi"
	bind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	common "github.com/ethereum/go-ethereum/common"
	goTypes "github.com/ethereum/go-ethereum/core/types"
	ethclient "github.com/ethereum/go-ethereum/ethclient"
	log "github.com/ethereum/go-ethereum/log"
	cli "github.com/urfave/cli"

	bindings "github.com/refcell/op-challenger/contracts/bindings"
	flags "github.com/refcell/op-challenger/flags"
	metrics "github.com/refcell/op-challenger/metrics"

	opBindings "github.com/ethereum-optimism/optimism/op-bindings/bindings"
	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	sources "github.com/ethereum-optimism/optimism/op-node/sources"
	oplog "github.com/ethereum-optimism/optimism/op-service/log"
	oppprof "github.com/ethereum-optimism/optimism/op-service/pprof"
	oprpc "github.com/ethereum-optimism/optimism/op-service/rpc"
	txmgr "github.com/ethereum-optimism/optimism/op-service/txmgr"
)

var supportedL2OutputVersion = eth.Bytes32{}

// Main is the entrypoint into the Challenger.
// This executes and blocks until the service exits.
func Main(version string, cliCtx *cli.Context) error {
	cfg := NewConfig(cliCtx)
	if err := cfg.Check(); err != nil {
		return fmt.Errorf("invalid CLI flags: %w", err)
	}

	l := oplog.NewLogger(cfg.LogConfig)
	m := metrics.NewMetrics("default")
	l.Info("Initializing Challenger")

	challengerConfig, err := NewChallengerConfigFromCLIConfig(cfg, l, m)
	if err != nil {
		l.Error("Unable to create the Challenger", "error", err)
		return err
	}

	challenger, err := NewChallenger(*challengerConfig, l, m)
	if err != nil {
		l.Error("Unable to create the Challenger", "error", err)
		return err
	}

	l.Info("Starting Challenger")
	ctx, cancel := context.WithCancel(context.Background())
	if err := challenger.Start(); err != nil {
		cancel()
		l.Error("Unable to start Challenger", "error", err)
		return err
	}
	defer challenger.Stop()

	l.Info("Challenger started")
	pprofConfig := cfg.PprofConfig
	if pprofConfig.Enabled {
		l.Info("starting pprof", "addr", pprofConfig.ListenAddr, "port", pprofConfig.ListenPort)
		go func() {
			if err := oppprof.ListenAndServe(ctx, pprofConfig.ListenAddr, pprofConfig.ListenPort); err != nil {
				l.Error("error starting pprof", "err", err)
			}
		}()
	}

	metricsCfg := cfg.MetricsConfig
	if metricsCfg.Enabled {
		l.Info("starting metrics server", "addr", metricsCfg.ListenAddr, "port", metricsCfg.ListenPort)
		go func() {
			if err := m.Serve(ctx, metricsCfg.ListenAddr, metricsCfg.ListenPort); err != nil {
				l.Error("error starting metrics server", err)
			}
		}()
		m.StartBalanceMetrics(ctx, l, challengerConfig.L1Client, challengerConfig.From)
	}

	rpcCfg := cfg.RPCConfig
	server := oprpc.NewServer(rpcCfg.ListenAddr, rpcCfg.ListenPort, version, oprpc.WithLogger(l))
	if err := server.Start(); err != nil {
		cancel()
		return fmt.Errorf("error starting RPC server: %w", err)
	}

	m.RecordInfo(version)
	m.RecordUp()

	interruptChannel := make(chan os.Signal, 1)
	signal.Notify(interruptChannel, []os.Signal{
		os.Interrupt,
		os.Kill,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	}...)
	<-interruptChannel
	cancel()

	return nil
}

// Challenger is responsible for disputing L2OutputOracle outputs
type Challenger struct {
	txMgr txmgr.TxManager
	wg    sync.WaitGroup
	done  chan struct{}
	log   log.Logger
	metr  metrics.Metricer

	from common.Address

	ctx    context.Context
	cancel context.CancelFunc

	l1Client *ethclient.Client

	rollupClient *sources.RollupClient

	l2ooContract     *opBindings.L2OutputOracleCaller
	l2ooContractAddr common.Address
	l2ooABI          *abi.ABI

	dgfContract     *bindings.MockDisputeGameFactoryCaller
	dgfContractAddr common.Address
	dgfABI          *abi.ABI

	adgABI *abi.ABI

	networkTimeout time.Duration
}

// NewChallengerFromCLIConfig creates a new L2 Output Submitter given the CLI Config
func NewChallengerFromCLIConfig(cfg CLIConfig, l log.Logger, m metrics.Metricer) (*Challenger, error) {
	proposerConfig, err := NewChallengerConfigFromCLIConfig(cfg, l, m)
	if err != nil {
		return nil, err
	}
	return NewChallenger(*proposerConfig, l, m)
}

// NewChallengerConfigFromCLIConfig creates the proposer config from the CLI config.
func NewChallengerConfigFromCLIConfig(cfg CLIConfig, l log.Logger, m metrics.Metricer) (*Config, error) {
	l2ooAddress, err := parseAddress(cfg.L2OOAddress)
	if err != nil {
		return nil, err
	}

	dgfAddress, err := parseAddress(cfg.DGFAddress)
	if err != nil {
		return nil, err
	}

	// Connect to L1 and L2 providers. Perform these last since they are the most expensive.
	ctx := context.Background()
	l1Client, err := dialEthClientWithTimeout(ctx, cfg.L1EthRpc)
	if err != nil {
		return nil, err
	}

	txManagerConfig, err := flags.NewTxManagerConfig(cfg.TxMgrConfig, l)
	if err != nil {
		return nil, err
	}
	txManager := txmgr.NewSimpleTxManager("challenger", l, txManagerConfig, l1Client)

	rollupClient, err := dialRollupClientWithTimeout(ctx, cfg.RollupRpc)
	if err != nil {
		return nil, err
	}

	return &Config{
		L2OutputOracleAddr: l2ooAddress,
		DisputeGameFactory: dgfAddress,
		NetworkTimeout:     txManagerConfig.NetworkTimeout,
		L1Client:           l1Client,
		RollupClient:       rollupClient,
		TxManager:          txManager,
		From:               txManagerConfig.From,
	}, nil
}

// NewChallenger creates a new Challenger
func NewChallenger(cfg Config, l log.Logger, m metrics.Metricer) (*Challenger, error) {
	ctx, cancel := context.WithCancel(context.Background())

	l2ooContract, err := opBindings.NewL2OutputOracleCaller(cfg.L2OutputOracleAddr, cfg.L1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	cCtx, cCancel := context.WithTimeout(ctx, cfg.NetworkTimeout)
	defer cCancel()
	version, err := l2ooContract.Version(&bind.CallOpts{Context: cCtx})
	if err != nil {
		cancel()
		return nil, err
	}
	log.Info("Connected to L2OutputOracle", "address", cfg.L2OutputOracleAddr, "version", version)

	parsed, err := opBindings.L2OutputOracleMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	dgfContract, err := bindings.NewMockDisputeGameFactoryCaller(cfg.DisputeGameFactory, cfg.L1Client)
	if err != nil {
		cancel()
		return nil, err
	}

	dgfAbi, err := bindings.MockDisputeGameFactoryMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	adgAbi, err := bindings.MockAttestationDisputeGameMetaData.GetAbi()
	if err != nil {
		cancel()
		return nil, err
	}

	return &Challenger{
		txMgr:  cfg.TxManager,
		done:   make(chan struct{}),
		log:    l,
		ctx:    ctx,
		cancel: cancel,
		metr:   m,

		from: cfg.From,

		l1Client: cfg.L1Client,

		rollupClient: cfg.RollupClient,

		l2ooContract:     l2ooContract,
		l2ooContractAddr: cfg.L2OutputOracleAddr,
		l2ooABI:          parsed,

		dgfContract:     dgfContract,
		dgfContractAddr: cfg.DisputeGameFactory,
		dgfABI:          dgfAbi,

		adgABI: adgAbi,

		networkTimeout: cfg.NetworkTimeout,
	}, nil
}

func (c *Challenger) Start() error {
	c.wg.Add(1)
	go c.loop()
	return nil
}

func (c *Challenger) Stop() {
	c.cancel()
	close(c.done)
	c.wg.Wait()
}

// ValidateOutput checks that a given output is expected via a trusted rollup node rpc.
// It returns: if the output is correct, error
func (c *Challenger) ValidateOutput(ctx context.Context, l2BlockNumber *big.Int, expected eth.Bytes32) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cancel()
	output, err := c.rollupClient.OutputAtBlock(ctx, l2BlockNumber.Uint64())
	if err != nil {
		c.log.Error("failed to fetch output for l2BlockNumber %d: %w", l2BlockNumber, err)
		return true, err
	}
	if output.Version != supportedL2OutputVersion {
		c.log.Error("unsupported l2 output version: %s", output.Version)
		return true, errors.New("unsupported l2 output version")
	}
	// If the block numbers don't match, we should try to fetch the output again
	if output.BlockRef.Number != l2BlockNumber.Uint64() {
		c.log.Error("invalid blockNumber: next blockNumber is %v, blockNumber of block is %v", l2BlockNumber, output.BlockRef.Number)
		return true, errors.New("invalid blockNumber")
	}
	if output.OutputRoot == expected {
		c.metr.RecordValidOutput(output.BlockRef)
	}
	return output.OutputRoot != expected, nil
}

// sendTransaction creates & sends transactions through the underlying transaction manager.
func (c *Challenger) sendTransaction(ctx context.Context, output common.Hash) error {
	// Get the adg address from the dgf
	cCtx, cCancel := context.WithTimeout(ctx, c.networkTimeout)
	defer cCancel()
	adgAddr, err := c.dgfContract.Games(&bind.CallOpts{Context: cCtx}, 1, output, []byte{})
	if err != nil {
		return err
	}
	// adgContract, err := bindings.NewMockAttestationDisputeGameCaller(adgAddr, c.l1Client)
	// if err != nil {
	// 	return err
	// }
	// TODO: Create an eip-712 signature for the contested output
	data, err := c.adgABI.Pack("challenge", output)
	if err != nil {
		return err
	}
	receipt, err := c.txMgr.Send(ctx, txmgr.TxCandidate{
		TxData:   data,
		To:       adgAddr,
		GasLimit: 0,
		From:     c.from,
	})
	if err != nil {
		return err
	}
	c.log.Info("challenger tx successfully published", "tx_hash", receipt.TxHash)
	return nil
}

// loop is responsible for creating & submitting the next outputs
func (c *Challenger) loop() {
	defer c.wg.Done()

	ctx := c.ctx

	// Listen for `OutputProposed` events from the L2 Output Oracle contract
	event := c.l2ooABI.Events["OutputProposed"]
	query := goEth.FilterQuery{
		Topics: [][]common.Hash{
			{event.ID},
		},
	}

	logs := make(chan goTypes.Log)
	sub, err := c.l1Client.SubscribeFilterLogs(context.Background(), query, logs)
	if err != nil {
		c.log.Error("failed to subscribe to logs", "err", err)
		return
	}

	for {
		select {
		case err := <-sub.Err():
			c.log.Error("failed to subscribe to logs", "err", err)
			return

		case vLog := <-logs:
			/*
				Event is encoded as:
					bytes32 indexed outputRoot,
					uint256 indexed l2OutputIndex,
					uint256 indexed l2BlockNumber,
					uint256 l1Timestamp
			*/
			l2BlockNumber := new(big.Int).SetBytes(vLog.Topics[3][:])
			expected := vLog.Topics[1]
			c.log.Info("Validating output", "l2BlockNumber", l2BlockNumber, "outputRoot", expected.Hex())
			isValid, err := c.ValidateOutput(ctx, l2BlockNumber, eth.Bytes32(common.Hex2BytesFixed(expected.Hex(), 32)))
			if err != nil || isValid {
				break
			}

			// Submit a challenge
			c.metr.RecordInvalidOutput(
				eth.L2BlockRef{
					Hash:   vLog.Topics[0],
					Number: l2BlockNumber.Uint64(),
				},
			)
			cCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			if err := c.sendTransaction(cCtx, expected); err != nil {
				c.log.Error("Failed to challenge transaction", "err", err)
				cancel()
				break
			}
			c.metr.RecordChallengeSent(l2BlockNumber, expected)
			cancel()
		case <-c.done:
			return
		}
	}
}
