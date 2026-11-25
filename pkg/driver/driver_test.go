package driver

import (
	"context"
	"testing"

	"github.com/lim123123123/concolic-testing/pkg/driver/expression"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type triangleInputKey struct{}

func triangleSetup(ctx context.Context) context.Context {
	in := [3]int{Int(ctx, "a"), Int(ctx, "b"), Int(ctx, "c")}
	return context.WithValue(ctx, triangleInputKey{}, in)
}

// TODO: ofc this ObservePathCondition calls should be placed automatically somehow
func triangleProgram(ctx context.Context) string {
	d := ctx.Value(DriverKey).(*Driver)
	in := ctx.Value(triangleInputKey{}).([3]int)
	a, b, c := in[0], in[1], in[2]
	if a == b {
		d.ObservePathCondition(expression.Equal(expression.Variable("a"), expression.Variable("b")))
		if b == c {
			d.ObservePathCondition(expression.Equal(expression.Variable("b"), expression.Variable("c")))
			return "equilateral"
		}
		d.ObservePathCondition(expression.NotEqual(expression.Variable("b"), expression.Variable("c")))
		return "isosceles1"
	}
	d.ObservePathCondition(expression.NotEqual(expression.Variable("a"), expression.Variable("b")))
	if a == c {
		d.ObservePathCondition(expression.Equal(expression.Variable("a"), expression.Variable("c")))
		return "isosceles2"
	}
	d.ObservePathCondition(expression.NotEqual(expression.Variable("a"), expression.Variable("c")))
	if b == c {
		d.ObservePathCondition(expression.Equal(expression.Variable("b"), expression.Variable("c")))
		return "isosceles3"
	}
	d.ObservePathCondition(expression.NotEqual(expression.Variable("b"), expression.Variable("c")))

	return "scalene"
}

func TestTriangle(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	driver := NewDriver(context.Background(), logger)

	var outputsSeen []string
	err := driver.Explore(triangleSetup, func(ctx context.Context) {
		outputsSeen = append(outputsSeen, triangleProgram(ctx))
	})
	require.NoError(t, err)

	allOutputs := []string{"equilateral", "isosceles1", "isosceles2", "isosceles3", "scalene"}
	require.ElementsMatch(t, allOutputs, lo.Uniq(outputsSeen))
}
