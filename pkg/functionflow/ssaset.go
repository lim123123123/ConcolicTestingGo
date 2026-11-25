package functionflow

import (
	"iter"
	"slices"
	"strings"

	"golang.org/x/tools/go/ssa"
)

type ssaValueSet struct {
	set map[ssa.Value]struct{}
}

func (svs *ssaValueSet) updateIfNil() {
	if svs.set == nil {
		svs.set = make(map[ssa.Value]struct{})
	}
}

func (svs *ssaValueSet) AddValue(val ssa.Value) {
	svs.updateIfNil()

	svs.set[val] = struct{}{}
}

func (svs *ssaValueSet) Merge(other *ssaValueSet) *ssaValueSet {
	svs.updateIfNil()

	if other == nil {
		return svs
	}
	for val := range other.set {
		svs.set[val] = struct{}{}
	}

	return svs
}

// All iterates over the values in the set in unspecified order.
// It is safe to call on a nil set.
func (svs *ssaValueSet) All() iter.Seq[ssa.Value] {
	return func(yield func(ssa.Value) bool) {
		if svs == nil {
			return
		}
		for val := range svs.set {
			if !yield(val) {
				return
			}
		}
	}
}

// Contains reports whether val is in the set. It is safe to call on a nil set.
func (svs *ssaValueSet) Contains(val ssa.Value) bool {
	if svs == nil {
		return false
	}
	_, ok := svs.set[val]
	return ok
}

// Diff returns a new set with the values of a that are not in b.
// Either argument may be nil, and neither is modified.
func diff(a, b *ssaValueSet) *ssaValueSet {
	result := &ssaValueSet{}
	for val := range a.All() {
		if b.Contains(val) {
			continue
		}
		result.AddValue(val)
	}
	return result
}

// Len returns the number of values in the set. It is safe to call on a nil set.
func (svs *ssaValueSet) Len() int {
	if svs == nil {
		return 0
	}
	return len(svs.set)
}

// String prints the names of the values in sorted order, e.g. {t1, x}.
// It is safe to call on a nil set.
func (svs *ssaValueSet) String() string {
	names := make([]string, 0, svs.Len())
	for val := range svs.All() {
		names = append(names, val.Name())
	}
	slices.Sort(names)

	return "{" + strings.Join(names, ", ") + "}"
}
