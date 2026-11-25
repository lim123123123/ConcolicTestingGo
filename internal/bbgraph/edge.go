package bbgraph

import "gonum.org/v1/gonum/graph"

type graphEdge struct {
	from, to graphNode
}

var _ graph.Edge = graphEdge{}

// From implements [graph.Edge].
func (e graphEdge) From() graph.Node {
	return e.from
}

// To implements [graph.Edge].
func (e graphEdge) To() graph.Node {
	return e.to
}

// ReversedEdge implements [graph.Edge].
func (e graphEdge) ReversedEdge() graph.Edge {
	return graphEdge{from: e.to, to: e.from}
}
