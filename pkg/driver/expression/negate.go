package expression

import (
	"fmt"

	"github.com/ezrantn/go-z3"
)

type negate struct {
	arg Expression
}

var _ Expression = &negate{}

func Negate(arg Expression) Expression {
	return &negate{
		arg: arg,
	}
}

func (v *negate) String() string {
	return fmt.Sprintf("!%s", v.arg)
}

func (v *negate) ToZ3(ctx z3.Context) *z3.Expr {
	return ctx.Not(v.arg.ToZ3(ctx))
}

func (v *negate) ExtractValues(model *z3.Model) []NamedValue {
	return v.arg.ExtractValues(model)
}
