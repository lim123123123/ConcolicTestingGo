package expression

import (
	"fmt"

	"github.com/ezrantn/go-z3"
)

const (
	OpEqual BinaryOp = iota
	OpNotEqual
)

var binaryOpSymbols = map[BinaryOp]string{
	OpEqual:    "==",
	OpNotEqual: "!=",
}

type BinaryOp int

type binaryOpExp struct {
	Left  Expression
	Right Expression
	Op    BinaryOp
}

var _ Expression = &binaryOpExp{}

func Equal(left, right Expression) Expression {
	return &binaryOpExp{
		Left:  left,
		Right: right,
		Op:    OpEqual,
	}
}

func NotEqual(left, right Expression) Expression {
	return &binaryOpExp{
		Left:  left,
		Right: right,
		Op:    OpNotEqual,
	}
}

func (b *binaryOpExp) ToZ3(ctx z3.Context) *z3.Expr {
	left := b.Left.ToZ3(ctx)
	right := b.Right.ToZ3(ctx)

	switch b.Op {
	case OpEqual:
		return ctx.Eq(left, right)
	case OpNotEqual:
		return ctx.Not(ctx.Eq(left, right))
	default:
		panic("unsupported op")
	}
}

func (b *binaryOpExp) ExtractValues(model *z3.Model) []NamedValue {
	return append(b.Left.ExtractValues(model), b.Right.ExtractValues(model)...)
}

func (b *binaryOpExp) String() string {
	return fmt.Sprintf("(%s %s %s)", b.Left, binaryOpSymbols[b.Op], b.Right)
}
