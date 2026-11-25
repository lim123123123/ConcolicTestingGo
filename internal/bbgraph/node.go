package bbgraph

import (
	"golang.org/x/tools/go/ssa"
	"gonum.org/v1/gonum/graph"
)

// graphNode is a basic block of the analyzed function, or the virtual exit node.
type graphNode struct {
	id    int64
	block *ssa.BasicBlock // nil for the virtual exit
}

var _ graph.Node = graphNode{}

// ID implements [graph.Node].
func (n graphNode) ID() int64 {
	return n.id
}
