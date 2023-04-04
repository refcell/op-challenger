package metrics

import (
	eth "github.com/ethereum-optimism/optimism/op-node/eth"
	opmetrics "github.com/ethereum-optimism/optimism/op-service/metrics"
	types "github.com/ethereum/go-ethereum/core/types"
)

type noopMetrics struct {
	opmetrics.NoopRefMetrics
}

var NoopMetrics Metricer = new(noopMetrics)

func (*noopMetrics) RecordInfo(version string) {}
func (*noopMetrics) RecordUp()                 {}

func (*noopMetrics) RecordValidOutput(l2ref eth.L2BlockRef)   {}
func (*noopMetrics) RecordInvalidOutput(l2ref eth.L2BlockRef) {}
func (*noopMetrics) RecordL1GasFee(receipt *types.Receipt)    {}
