package functionflow

import (
	"fmt"
	"slices"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/lim123123123/concolic-testing/internal/bbgraph"
)

type FunctionFlowAnalyzer struct {
	config         functionFlowAnalyzerConfig
	analyzedFunc   *ssa.Function
	controlFlowDep map[int]*ssaValueSet // block id -> blocks on which it depends
	ipdoms         map[int]int          // block id -> immediate post dominator block id
	ipdomsReversed map[int]*ssaValueSet // block -> values from the blocks that it immediately post dominates
	flow           map[ssa.Value]*ssaValueSet
}

func NewAnalyzer(pathToPkg, funcName string, opts ...FunctionFlowAnalyzerOption) (*FunctionFlowAnalyzer, error) {
	config := functionFlowAnalyzerConfig{}
	for _, opt := range opts {
		opt(&config)
	}

	pkg, err := buildPackage(pathToPkg)
	if err != nil {
		return nil, fmt.Errorf("can't build package: %w", err)
	}

	ssaFunc, err := buildSSA(pkg, funcName)
	if err != nil {
		return nil, fmt.Errorf("can't build ssa: %w", err)
	}
	if config.DumpSSAToFile != "" {
		if err := dumpSSA(ssaFunc, config.DumpSSAToFile); err != nil {
			return nil, fmt.Errorf("can't dump ssa: %w", err)
		}
	}

	ipdoms := bbgraph.NewFromSSAFunc(ssaFunc).ImmediatePostDominators()

	return &FunctionFlowAnalyzer{
		config:         config,
		analyzedFunc:   ssaFunc,
		controlFlowDep: map[int]*ssaValueSet{},
		ipdoms:         ipdoms,
		ipdomsReversed: calculateReversedIpdoms(ipdoms, ssaFunc),
		flow:           map[ssa.Value]*ssaValueSet{},
	}, nil
}

// calculateReversedIpdoms maps a block index to the conditions of the if statements
// that end the blocks it immediately post-dominates. These are the controllers
// that stop controlling execution once the block is reached.
func calculateReversedIpdoms(ipdoms map[int]int, ssaFunc *ssa.Function) map[int]*ssaValueSet {
	reversed := map[int]*ssaValueSet{}
	for _, bb := range ssaFunc.Blocks {
		ifInstr, ok := bb.Instrs[len(bb.Instrs)-1].(*ssa.If)
		if !ok {
			continue
		}
		ipdom, ok := ipdoms[bb.Index]
		if !ok {
			continue
		}
		if reversed[ipdom] == nil {
			reversed[ipdom] = &ssaValueSet{}
		}
		reversed[ipdom].AddValue(ifInstr.Cond)
	}

	return reversed
}

// ctrlOf returns the (possibly empty) set of controllers of the block.
func (analyzer *FunctionFlowAnalyzer) ctrlOf(blockID int) *ssaValueSet {
	if analyzer.controlFlowDep[blockID] == nil {
		analyzer.controlFlowDep[blockID] = &ssaValueSet{}
	}
	return analyzer.controlFlowDep[blockID]
}

// This thing is all about reachability. Each statement should be reachable thorough
// one of the inputs. So for each statement I must calculate the set of variables on which it depends
// and then one of the inputs should make all these variables symbolic.
// Depends there means that considering the path to the given statement - we take all the if
// statements on which we depends, and take all variables recursively that affect the conditions of these
// ifs
func (analyzer *FunctionFlowAnalyzer) FindFlow() map[ssa.Value]*ssaValueSet {
	entry := analyzer.analyzedFunc.Blocks[0]
	visited := map[int]bool{entry.Index: true}
	analyzer.exploreBlock(entry, visited)
	return analyzer.flow
}

// TODO: input params could be computed automatically
func (analyzer *FunctionFlowAnalyzer) ExtractBlocks(inputs []*ssa.Parameter, flow map[ssa.Value]*ssaValueSet) [][]*ssa.Parameter {
	return mergeBlocks(inputs, flow)
}

func (analyzer *FunctionFlowAnalyzer) getOrCreateFlow(v ssa.Value) *ssaValueSet {
	if analyzer.flow[v] == nil {
		analyzer.flow[v] = &ssaValueSet{}
		analyzer.flow[v].AddValue(v)
	}
	return analyzer.flow[v]
}

// exploreBlock updates flow for every instruction of blk under the controllers
// stored in controlFlowDep[blk], then recurses into all successors, so both arms
// of an if are explored. A visited block is explored again only if new controllers
// reached it, which guarantees termination on loops because the sets only grow.
func (analyzer *FunctionFlowAnalyzer) exploreBlock(
	blk *ssa.BasicBlock,
	visited map[int]bool,
) {
	ctrl := analyzer.ctrlOf(blk.Index)

	// Each basic block could have only one if - in the end of the block
	// So in the end of the block we need to calculate the ctrl dependencies
	// for the next one or possibly two next ones, if there is an if in the end
	outgoing := (&ssaValueSet{}).Merge(ctrl)

	for _, instr := range blk.Instrs {
		switch tInstr := instr.(type) {
		case *ssa.BinOp:
			// Add the data dependencies to the flow, skip consts
			// TODO: separate function for this const check + merge
			if _, ok := tInstr.X.(*ssa.Const); !ok {
				// The value isn't a constant, so add the data dependency
				analyzer.getOrCreateFlow(tInstr).Merge(analyzer.getOrCreateFlow(tInstr.X))
			}

			if _, ok := tInstr.Y.(*ssa.Const); !ok {
				// The value isn't a constant, so add the data dependency
				analyzer.getOrCreateFlow(tInstr).Merge(analyzer.getOrCreateFlow(tInstr.Y))
			}

			for dep := range ctrl.All() {
				analyzer.getOrCreateFlow(tInstr).Merge(analyzer.getOrCreateFlow(dep))
			}
		case *ssa.If:
			for dep := range ctrl.All() {
				analyzer.getOrCreateFlow(tInstr.Cond).Merge(analyzer.getOrCreateFlow(dep))
			}
			outgoing.AddValue(tInstr.Cond)
		case *ssa.Phi:
			// The phi node is a bit special here, in essence it depends on data dependencies
			// of its arguments, but in practice they could be constants. In this case the dependency
			// is carried by the control flow dependencies of all paths
			for edgeIdx, valueForThisBlock := range tInstr.Edges {
				// Edges[i] is the value arriving from Preds[i]
				prevBlockID := blk.Preds[edgeIdx].Index
				if _, ok := valueForThisBlock.(*ssa.Const); !ok {
					// The value isn't a constant, so add the data dependency
					analyzer.getOrCreateFlow(tInstr).Merge(analyzer.getOrCreateFlow(valueForThisBlock))
				}

				// Now we need to add the flow from ctrl dependencies
				for ctrlDep := range analyzer.ctrlOf(prevBlockID).All() {
					analyzer.getOrCreateFlow(tInstr).Merge(analyzer.getOrCreateFlow(ctrlDep))
				}

				// ctrlOf(pred) does not contain the condition of the if that ends pred itself:
				// it is only handed to the successors. So an edge that leaves an if block directly
				// into this block could lose that condition. In structured code a sibling edge
				// through an arm block usually carries it, but if such a phi shows up, also merge
				// the flow of the Cond of the last if instruction of pred.
			}
		case *ssa.Jump:
			// No flow of its own: the controllers for the target are computed
			// in the successor loop below.
		case *ssa.Alloc:
			analyzer.handleAllocInstr(tInstr, ctrl)
		case *ssa.Store:
			analyzer.handleStoreInstr(tInstr, ctrl)
		case *ssa.UnOp:
			analyzer.handleUnOp(tInstr, ctrl)
		case *ssa.FieldAddr:
			analyzer.handleFieldAddrInstr(tInstr, ctrl)
		default:
			analyzer.config.Logger.Errorf("unsupported instr: %s", instr)
		}

		analyzer.config.Logger.Debugf(
			"flow after instr %s alloc: %s",
			instr,
			formatFlow(analyzer.flow),
		)
	}

	// Calculate the ctrl for the next blocks
	for _, succ := range blk.Succs {
		succCtrl := analyzer.ctrlOf(succ.Index)
		before := succCtrl.Len()
		// We must add all our ctrl dependencies, except the ones for which the next block is ipdom
		// All out dependencies, except these for which the next one is ipdom
		// need to find
		succCtrl.Merge(diff(outgoing, analyzer.ipdomsReversed[succ.Index]))
		if visited[succ.Index] && succCtrl.Len() == before {
			continue
		}
		// Cycle is gonna point to itself
		visited[succ.Index] = true
		analyzer.exploreBlock(succ, visited)
	}
}

// formatFlow prints the flow map one entry per line in sorted order, e.g. t2 -> {t1, x}.
func formatFlow(flow map[ssa.Value]*ssaValueSet) string {
	lines := make([]string, 0, len(flow))
	for val, set := range flow {
		lines = append(lines, fmt.Sprintf("%s -> %s", val.Name(), set))
	}
	slices.Sort(lines)

	return strings.Join(lines, "\n")
}
