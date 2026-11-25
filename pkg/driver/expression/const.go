package expression

import (
	"fmt"

	"github.com/ezrantn/go-z3"
)

type constant struct {
	value int
}

var _ Expression = &constant{}

func Constant(value int) Expression {
	return &constant{
		value: value,
	}
}

// ExtractValues implements [Expression].
func (c *constant) ExtractValues(*z3.Model) []NamedValue {
	return []NamedValue{}
}

// String implements [Expression].
func (c *constant) String() string {
	return fmt.Sprintf("%d", c.value)
}

// ToZ3 implements [Expression].
func (c *constant) ToZ3(ctx z3.Context) *z3.Expr {
	return ctx.Int(c.value, ctx.IntSort())
}
