package expression

import (
	"fmt"

	"github.com/ezrantn/go-z3"
)

type NamedValue struct {
	Name  string // parameter
	Value int
}

// TODO: type assert everywhere
type Expression interface {
	fmt.Stringer
	ToZ3(z3.Context) *z3.Expr
	ExtractValues(*z3.Model) []NamedValue
}
