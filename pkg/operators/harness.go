package operators

import (
	"context"
	"fmt"
	"strings"

	"github.com/lim123123123/concolic-testing/pkg/driver"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type clientKeyType int

var ClientKey clientKeyType = 1

type crReqKeyType int

var CRReqKey crReqKeyType = 1

type crKeyType int

var CRKey crKeyType = 1

var reconciliationThreshold = 10

type OperatorExplorer[CR comparable] struct {
	setUpFunc             driver.SetupFunc
	reconcilerConstructor func(client.Client, *runtime.Scheme, *driver.Driver) Reconciler
	scheme                *runtime.Scheme
	seenOutputs           map[CR]client.Client
}

type Reconciler interface {
	Reconcile(request reconcile.Request) (reconcile.Result, error)
}

func NewOperatorExplorer[CR comparable](
	setUpFunc driver.SetupFunc,
	reconcilerConstructor func(client.Client, *runtime.Scheme, *driver.Driver) Reconciler,
	scheme *runtime.Scheme,
) *OperatorExplorer[CR] {
	return &OperatorExplorer[CR]{
		setUpFunc:             setUpFunc,
		reconcilerConstructor: reconcilerConstructor,
		scheme:                scheme,
		seenOutputs:           map[CR]client.Client{},
	}
}

// Maybe separate everything to two functions? One takes arguments and do sth. The second one - runs?

// I need to generate arguments
// Set up them to the cr
// Set up the cr to the fake client
// run reconciliation
// Generate arguments from scratch again
// Update the cr
// Run reconciliation
// repeat n times
// Run reconciliation only with last args
// Compare outputs

//

// I could symbolically execute one function and it takes all generated args as int arguments
// Separate args and function in the driver

// Exploration starts
// The bound is 1
// Before calling reconcile - I create a new client with the crd and only it
// I execute reconcile until it returns the value that says to do so
// I save the resulting cluster state as a client somewhere

// If the bound is 2 -
// I generate the resulting cluster states and then in setup function I need to explore all paths strating with each possible previous state

// func (oe *OperatorExplorer[CR]) exploreWithGivenClient(client client.Client) {}

func executeReconciler(req reconcile.Request, reconciler Reconciler) error {

	for i := 0; i < reconciliationThreshold; i++ {
		res, err := reconciler.Reconcile(req)
		if err != nil {
			return fmt.Errorf("error during reconciliation: %w", err)
		}

		// TODO: how should we handle req.RequeueAfter here?
		// Operator may use it to perform reconciliation once in a while
		// If it doesn't subscribed to all created resources
		if !res.Requeue {
			return nil
		}
	}

	return fmt.Errorf("too many reconciliations are required")
}

// Setup func has to set up the CRReqKey with the name and the namespace
// Setup func has to set up CRKey with the resulting CR
// If error is returned that means that something went wrong during the execution
// If the test itself is failing the statistics shows it (TODO: add another return param)
func (oe *OperatorExplorer[CR]) Explore() error {
	driver := driver.NewDriver(context.Background(), zap.S())
	fmt.Print("driver in harness")
	fmt.Print(driver)

	// We need to set up a new client each time, so we need to create the client in setUpFunc
	// In some other cases we need to take a client from the previous iteration copy it and use it
	if err := driver.Explore(
		func(ctx context.Context) context.Context {
			client := fake.NewFakeClientWithScheme(oe.scheme)

			return oe.setUpFunc(context.WithValue(ctx, ClientKey, client))
		}, func(ctx context.Context) {
			client := ctx.Value(ClientKey).(client.Client)

			if err := executeReconciler(
				ctx.Value(CRReqKey).(reconcile.Request),
				oe.reconcilerConstructor(client, oe.scheme, driver),
			); err != nil {
				// TODO: logger
				panic(fmt.Errorf("error during initial reconciliation: %s", err.Error()))
			}

			cr := ctx.Value(CRKey).(CR)
			oe.seenOutputs[cr] = client
		},
	); err != nil {
		return fmt.Errorf("error during initial exploration: %w", err)
	}

	for firstCfg, state := range oe.seenOutputs {
		if err := driver.Explore(
			func(ctx context.Context) context.Context {
				// Every run mutates its client, so each one starts from a fresh copy of the state
				// reached by reconciling the first CR
				stateCopy, err := copyClient(state, oe.scheme)
				if err != nil {
					panic(fmt.Errorf("copy client: %w", err))
				}
				return oe.setUpFunc(context.WithValue(ctx, ClientKey, stateCopy))
			}, func(ctx context.Context) {
				client := ctx.Value(ClientKey).(client.Client)

				if err := executeReconciler(
					ctx.Value(CRReqKey).(reconcile.Request),
					oe.reconcilerConstructor(client, oe.scheme, driver),
				); err != nil {
					// TODO: logger
					panic(fmt.Errorf("error during second reconciliation: %s", err.Error()))
				}

				cr := ctx.Value(CRKey).(CR)
				// Reconcile(Reconcile(0, firstCfg), cr) must equal Reconcile(0, cr)
				expected, ok := oe.seenOutputs[cr]
				if !ok {
					zap.S().Warnf("no state reconciled from scratch for CR %v, skipping the comparison", cr)
					return
				}

				diffs, err := diffClusters(oe.scheme, expected, client)
				if err != nil {
					panic(fmt.Errorf("compare clusters: %w", err))
				}
				if len(diffs) != 0 {
					panic(fmt.Errorf("Reconcile(Reconcile(0, %v), %v) != Reconcile(0, %v):\n%s", firstCfg, cr, cr, strings.Join(diffs, "\n")))
				}
			},
		); err != nil {
			return fmt.Errorf("error during second exploration: %w", err)
		}
	}

	// TODO: pass the errors everywhere
	return nil
}
