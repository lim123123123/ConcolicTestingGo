package functionflow

import (
	"go.chromium.org/luci/common/data/disjointset"
	"golang.org/x/tools/go/ssa"
)

// mergeBlocks implements the merge step of FlowTest. It starts with one block per
// input and, for every flow set, merges all the inputs in that set into a single block.
// Values in the flow sets that are not inputs are ignored. The blocks are ordered
// by size (largest first), and the inputs inside a block by their position in inputs.
func mergeBlocks(inputs []*ssa.Parameter, flow map[ssa.Value]*ssaValueSet) [][]*ssa.Parameter {
	inputIdx := make(map[ssa.Value]int, len(inputs))
	for i, input := range inputs {
		inputIdx[input] = i
	}

	blocks := disjointset.New(len(inputs))
	for _, influencers := range flow {
		first := -1
		for val := range influencers.All() {
			idx, isInput := inputIdx[val]
			if !isInput {
				continue
			}
			if first == -1 {
				first = idx
				continue
			}
			blocks.Merge(first, idx)
		}
	}

	result := make([][]*ssa.Parameter, 0, blocks.Count())
	for _, indices := range blocks.SortedSets() {
		block := make([]*ssa.Parameter, 0, len(indices))
		for _, idx := range indices {
			block = append(block, inputs[idx])
		}
		result = append(result, block)
	}

	return result
}
