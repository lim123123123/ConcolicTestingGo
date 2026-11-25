package expression

import (
	"fmt"

	"github.com/ezrantn/go-z3"
)

type variable struct {
	name         string
	previousExpr *z3.Expr // TODO: the design of all these z3 bindings makes me ask some questions
}

var _ Expression = &variable{}

func Variable(name string) Expression {
	return &variable{
		name: name,
	}
}

func (v *variable) ToZ3(ctx z3.Context) *z3.Expr {
	// TODO: find normal bindings
	v.previousExpr = ctx.Const(v.name, ctx.IntSort())
	return v.previousExpr
}

func (v *variable) ExtractValues(model *z3.Model) []NamedValue {
	intRepresentation, ok := model.EvalInt64(v.previousExpr)
	if !ok {
		panic(fmt.Sprintf("model value of %q is not an int64 numeral", v.name))
	}

	return []NamedValue{
		{
			Name:  v.name,
			Value: int(intRepresentation),
		},
	}
}

func (v *variable) String() string {
	return v.name
}
