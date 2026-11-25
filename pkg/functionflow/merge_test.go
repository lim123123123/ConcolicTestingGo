package functionflow

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/ssa"
)

func TestMergeBlocks(t *testing.T) {
	x, y, z := &ssa.Parameter{}, &ssa.Parameter{}, &ssa.Parameter{}
	temp := &ssa.Parameter{} // stands for an intermediate value, not an input
	inputs := []*ssa.Parameter{x, y, z}

	setOf := func(vals ...ssa.Value) *ssaValueSet {
		set := &ssaValueSet{}
		for _, v := range vals {
			set.AddValue(v)
		}
		return set
	}

	tests := []struct {
		name string
		flow map[ssa.Value]*ssaValueSet
		want [][]*ssa.Parameter
	}{
		{
			name: "no dependencies keeps singletons",
			flow: map[ssa.Value]*ssaValueSet{temp: setOf(temp, x)},
			want: [][]*ssa.Parameter{{x}, {y}, {z}},
		},
		{
			name: "one set merges its inputs",
			flow: map[ssa.Value]*ssaValueSet{temp: setOf(temp, x, z)},
			want: [][]*ssa.Parameter{{x, z}, {y}},
		},
		{
			name: "overlapping sets merge transitively",
			flow: map[ssa.Value]*ssaValueSet{
				temp: setOf(x, y),
				x:    setOf(y, z),
			},
			want: [][]*ssa.Parameter{{x, y, z}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, mergeBlocks(inputs, tt.flow))
		})
	}
}
