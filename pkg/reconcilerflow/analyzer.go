package reconcilerflow

import "github.com/lim123123123/concolic-testing/pkg/functionflow"

type ReconcilerFlowAnalyzer struct {
	pkgPath string
}

func NewReconcileFlowAnalyzer(pkgPath string) *ReconcilerFlowAnalyzer {
	return &ReconcilerFlowAnalyzer{
		pkgPath: pkgPath,
	}
}

func (rfa *ReconcilerFlowAnalyzer) ComputeBlocks() ([][]string, error) {
	fa, err := functionflow.NewAnalyzer(
		rfa.pkgPath, 
		"Reconcile",
		functionflow.WithDumpSSAToFile("/Users/lim123123123/projects/controllers-verification/concolic-testing/pkg/reconcilerflow/reconcilessa"),
	)
	if err != nil {
		return nil, err
	}

	fa.FindFlow()
	panic("")
	// fa.ExtractBlocks()
}
