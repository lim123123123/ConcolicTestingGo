// Package bbgraph exposes the control flow graph of an ssa.Function, whose edges are
// implicit in ssa.BasicBlock.Succs, as an explicit gonum graph.
package bbgraph

import (
	"slices"

	"golang.org/x/tools/go/ssa"
	"gonum.org/v1/gonum/graph"
	"gonum.org/v1/gonum/graph/flow"
	"gonum.org/v1/gonum/graph/iterator"
)

// Graph is a control flow graph where node IDs are block indices. It has one extra
// virtual exit node, reachable from every block without successors (return or panic),
// so that the graph has a single exit, as the post-dominance relation requires.
type Graph struct {
	nodes map[int64]graphNode
	succs map[int64]map[int64]struct{}
	preds map[int64]map[int64]struct{}
	exit  graphNode
}

var _ graph.Directed = (*Graph)(nil)

// NewFromSSAFunc translates the blocks of ssaFunc and the edges between them.
// The virtual exit node has ID len(ssaFunc.Blocks).
func NewFromSSAFunc(ssaFunc *ssa.Function) *Graph {
	g := &Graph{
		nodes: map[int64]graphNode{},
		succs: map[int64]map[int64]struct{}{},
		preds: map[int64]map[int64]struct{}{},
		exit:  graphNode{id: int64(len(ssaFunc.Blocks))},
	}
	g.nodes[g.exit.id] = g.exit

	for _, bb := range ssaFunc.Blocks {
		g.nodes[int64(bb.Index)] = graphNode{id: int64(bb.Index), block: bb}
	}
	for _, bb := range ssaFunc.Blocks {
		if len(bb.Succs) == 0 {
			g.addEdge(int64(bb.Index), g.exit.id)
		}
		for _, succ := range bb.Succs {
			g.addEdge(int64(bb.Index), int64(succ.Index))
		}
	}

	return g
}

// ExitID is the ID of the virtual exit node.
func (g *Graph) ExitID() int {
	return int(g.exit.id)
}

// ImmediatePostDominators maps a block index to the index of its immediate
// post-dominator, which is ExitID() for blocks post-dominated only by the exit.
// Blocks from which the exit is unreachable (infinite loops) have no entry.
func (g *Graph) ImmediatePostDominators() map[int]int {
	reversed := g.reversed()
	// Post-dominators of g are the dominators of the reversed graph rooted at the exit.
	tree := flow.Dominators(reversed.exit, reversed)

	ipdoms := make(map[int]int, len(g.nodes))
	for id := range g.nodes {
		if id == g.exit.id {
			continue
		}
		if ipdom := tree.DominatorOf(id); ipdom != nil {
			ipdoms[int(id)] = int(ipdom.ID())
		}
	}

	return ipdoms
}

func (g *Graph) addEdge(fromID, toID int64) {
	if g.succs[fromID] == nil {
		g.succs[fromID] = map[int64]struct{}{}
	}
	if g.preds[toID] == nil {
		g.preds[toID] = map[int64]struct{}{}
	}
	g.succs[fromID][toID] = struct{}{}
	g.preds[toID][fromID] = struct{}{}
}

// reversed returns the graph with all edges flipped. The nodes are shared.
func (g *Graph) reversed() *Graph {
	return &Graph{
		nodes: g.nodes,
		succs: g.preds,
		preds: g.succs,
		exit:  g.exit,
	}
}

// nodesByID returns the nodes with the given IDs in ascending ID order,
// which keeps iteration deterministic.
func (g *Graph) nodesByID(ids map[int64]struct{}) graph.Nodes {
	sorted := make([]int64, 0, len(ids))
	for id := range ids {
		sorted = append(sorted, id)
	}
	slices.Sort(sorted)

	nodes := make([]graph.Node, 0, len(sorted))
	for _, id := range sorted {
		nodes = append(nodes, g.nodes[id])
	}

	return iterator.NewOrderedNodes(nodes)
}

// Node implements [graph.Graph].
func (g *Graph) Node(id int64) graph.Node {
	node, exist := g.nodes[id]
	if !exist {
		return nil
	}

	return node
}

// Nodes implements [graph.Graph].
func (g *Graph) Nodes() graph.Nodes {
	all := make(map[int64]struct{}, len(g.nodes))
	for id := range g.nodes {
		all[id] = struct{}{}
	}

	return g.nodesByID(all)
}

// From implements [graph.Graph].
func (g *Graph) From(id int64) graph.Nodes {
	return g.nodesByID(g.succs[id])
}

// To implements [graph.Directed].
func (g *Graph) To(id int64) graph.Nodes {
	return g.nodesByID(g.preds[id])
}

// HasEdgeFromTo implements [graph.Directed].
func (g *Graph) HasEdgeFromTo(uid, vid int64) bool {
	_, exist := g.succs[uid][vid]
	return exist
}

// HasEdgeBetween implements [graph.Graph].
func (g *Graph) HasEdgeBetween(xid, yid int64) bool {
	return g.HasEdgeFromTo(xid, yid) || g.HasEdgeFromTo(yid, xid)
}

// Edge implements [graph.Graph].
func (g *Graph) Edge(uid, vid int64) graph.Edge {
	if !g.HasEdgeFromTo(uid, vid) {
		return nil
	}

	return graphEdge{from: g.nodes[uid], to: g.nodes[vid]}
}
