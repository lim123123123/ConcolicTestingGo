package reconcilerflow

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeBlocks(t *testing.T) {
	rfa := NewReconcileFlowAnalyzer("./testdata/podWithConfigMapOp/reconciler")
	_, err := rfa.ComputeBlocks()
	require.NoError(t, err)
}
