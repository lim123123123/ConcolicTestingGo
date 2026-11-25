package driver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/ezrantn/go-z3"
	"github.com/lim123123123/concolic-testing/pkg/driver/expression"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

type driverKey string

var DriverKey driverKey = "driverKey"

var errUnsat = fmt.Errorf("provided conditions conjunction isn't solvable")

// SetupFunc builds the environment for one run, e.g. a fake client. It must return a context derived
// from the one it received. Symbolic integers are requested inside it with Int.
type SetupFunc func(ctx context.Context) context.Context

// TargetFunc is the function that is symbolically executed in the environment SetupFunc returned.
// It reports outputs to the caller through its own closure.
type TargetFunc func(ctx context.Context)

type Driver struct {
	currentPathConditions []expression.Expression
	z3Cfg                 *z3.Config
	logger                *zap.SugaredLogger
	baseContext           context.Context // Carries the driver, so Int and ObservePathCondition callers can find it
	currentValues         map[string]int  // Solver-chosen values of the symbolic ints for the current run
}

func NewDriver(ctx context.Context, logger *zap.SugaredLogger) *Driver {
	d := &Driver{
		z3Cfg:  z3.NewConfig(),
		logger: logger,
	}
	d.baseContext = context.WithValue(ctx, DriverKey, d)

	return d
}

// Int returns the value of the symbolic integer called name in the current run: the solver's choice if
// the path conditions mention it, 0 otherwise. Use the same name in Variable(name).
func Int(ctx context.Context, name string) int {
	d, ok := ctx.Value(DriverKey).(*Driver)
	if !ok {
		panic("driver.Int: context carries no driver, derive it from the context passed to setup")
	}
	return d.currentValues[name]
}

// current is the driver whose Explore is in progress. Code under test that can't receive the driver
// through its arguments (e.g. operator code that only sees a CR read back from a client) reads it
// with Current. It is not safe for concurrent explorations.
var current *Driver

// Current returns the driver exploring right now, or nil outside Explore.
func Current() *Driver {
	return current
}

// ObservePathCondition panics on a nil driver: a missing driver means the instrumented code ran
// outside Explore, and silently dropping its path conditions would make the exploration incomplete.
func (d *Driver) ObservePathCondition(exp expression.Expression) {
	if d == nil {
		panic("driver.ObservePathCondition: no driver set up, call it only while Explore is running")
	}
	d.currentPathConditions = append(d.currentPathConditions, exp)
}

func (d *Driver) generateNewPathWithZ3(conditions []expression.Expression) (map[string]int, error) {
	if len(conditions) == 0 {
		return nil, fmt.Errorf("Path conditions couldn't be empty")
	}

	// TODO: could we reuse this context?
	// Construct z3 solver query to generate actual function arguments
	z3Ctx := z3.NewContext(d.z3Cfg)
	expr := conditions[0].ToZ3(*z3Ctx)

	for _, cond := range conditions[1:] {
		expr = z3Ctx.And(expr, cond.ToZ3(*z3Ctx))
	}

	solver := z3Ctx.NewSolver()
	solver.Assert(expr)

	if solver.Check() {
		model := solver.GetModel()
		values := lo.FlatMap(conditions, func(cond expression.Expression, _ int) []expression.NamedValue {
			return cond.ExtractValues(model)
		})

		return lo.SliceToMap(values, func(v expression.NamedValue) (string, int) {
			return v.Name, v.Value
		}), nil
	}

	return nil, errUnsat
}

// formatPathConditions renders a path as a human-readable conjunction, e.g. "(x == y) && !(y != 3)".
func formatPathConditions(conditions []expression.Expression) string {
	parts := lo.Map(conditions, func(cond expression.Expression, _ int) string {
		return cond.String()
	})
	return strings.Join(parts, " && ")
}

// run executes setup with the given symbolic int values, then target in the environment setup returned.
// Path conditions observed during setup are discarded so only target's path is collected.
func (d *Driver) run(setup SetupFunc, target TargetFunc, values map[string]int) error {
	d.currentValues = values
	ctx := setup(d.baseContext)
	if ctx == nil || ctx.Value(DriverKey) != d {
		return fmt.Errorf("setup must return a context derived from the one it received")
	}

	d.currentPathConditions = []expression.Expression{}
	target(ctx)
	return nil
}

// explorePath is a depth-first search over path conditions: each condition at index >= bound is
// negated in turn (deepest first) and the resulting subtree is explored. Conditions before bound
// are fixed, because a caller already flipped one of them to reach this path.
func (d *Driver) explorePath(setup SetupFunc, target TargetFunc, path []expression.Expression, bound int) error {
	for i := len(path) - 1; i >= bound; i-- {
		// Clone so the negation doesn't mutate the caller's path
		candidate := append(slices.Clone(path[:i]), expression.Negate(path[i]))
		d.logger.Debugf("path conditions: %s", formatPathConditions(candidate))

		values, err := d.generateNewPathWithZ3(candidate)
		if errors.Is(err, errUnsat) {
			d.logger.Debugf("unsat path cond: %s", formatPathConditions(candidate))
			continue
		} else if err != nil {
			return err
		}

		d.logger.Debugf("call in explore with values: %v", values)
		if err := d.run(setup, target, values); err != nil {
			return err
		}

		newPath := slices.Clone(d.currentPathConditions)
		if len(newPath) <= i {
			// The run didn't reach the flipped branch, so the function is not deterministic in its arguments
			d.logger.Warnf("run diverged from predicted path, expected %d+ conditions, got: %s", i+1, formatPathConditions(newPath))
			continue
		}
		if err := d.explorePath(setup, target, newPath, i+1); err != nil {
			return err
		}
	}

	return nil
}

// Explore runs target on every path: each run first calls setup, where Int provides the solver-chosen
// values (all 0 on the initial run), then target in the environment setup returned.
func (d *Driver) Explore(setup SetupFunc, target TargetFunc) error {
	current = d
	defer func() { current = nil }()

	// Start the exploration process
	// Generate the initial path constraints
	d.logger.Debugf("Initial call, all symbolic ints are 0")
	if err := d.run(setup, target, nil); err != nil {
		return err
	}
	if len(d.currentPathConditions) == 0 {
		return fmt.Errorf("no path conditions were found on the initial run")
	}

	return d.explorePath(setup, target, slices.Clone(d.currentPathConditions), 0)
}
