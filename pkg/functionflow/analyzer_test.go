package functionflow

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"golang.org/x/tools/go/ssa"
)

func TestCalculateIdoms_Branching(t *testing.T) {
	if _, err := NewAnalyzer("./testdata/branching", "Branching"); err != nil {
		t.Fatalf("CalculateIdoms returned error: %v", err)
	}
}

func TestCalculateIdoms_FlowTest(t *testing.T) {
	if _, err := NewAnalyzer("./testdata/branching", "BreakFlowTest"); err != nil {
		t.Fatalf("CalculateIdoms returned error: %v", err)
	}
}

func TestImmediatePostDominators(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		// block index -> index of its immediate post-dominator,
		// the virtual exit has the index len(blocks)
		want map[int]int
	}{
		{
			name:     "Branching",
			funcName: "Branching",
			want:     map[int]int{0: 2, 1: 2, 2: 4, 3: 2},
		},
		{
			// the panic block is a second exit, so block 2 is not post-dominated by block 4
			name:     "BreakFlowTest",
			funcName: "BreakFlowTest",
			want:     map[int]int{0: 2, 1: 2, 2: 6, 3: 6, 4: 6, 5: 6},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			analyzer, err := NewAnalyzer("./testdata/branching", tt.funcName)
			require.NoError(t, err)

			require.Equal(t, tt.want, analyzer.ipdoms)
		})
	}
}

// flowTestCases holds, for each function from testdata, the flow and the blocks computed
// by hand. The flow is restricted to the inputs: a value is mapped to the sorted names
// of the function parameters that influence it, through data or control dependencies.
var flowTestCases = []struct {
	funcName   string
	wantFlow   map[string][]string
	wantBlocks [][]string
}{
	// {
	// 	// t6 = z + x + y mixes both inputs
	// 	funcName: "Branching",
	// 	wantFlow: map[string][]string{
	// 		"x": {"x"}, "y": {"y"},
	// 		"t0": {"x"},
	// 		"t1": {"x", "y"}, "t2": {"x", "y"},
	// 		"t3": {"x", "y"}, "t4": {"x", "y"}, "t5": {"x", "y"}, "t6": {"x", "y"},
	// 	},
	// 	wantBlocks: [][]string{{"x", "y"}},
	// },
	// {
	// 	// c is 2 only if x != 0, so the panic depends on both x and z
	// 	funcName: "BreakFlowTest",
	// 	wantFlow: map[string][]string{
	// 		"x": {"x"}, "z": {"z"},
	// 		"t0": {"x"}, "t1": {"x"}, "t2": {"x"}, "t3": {"x", "z"},
	// 	},
	// 	wantBlocks: [][]string{{"x", "z"}},
	// },
	// {
	// 	// no value mixes x with y, so the inputs stay in separate blocks
	// 	funcName: "Independent",
	// 	wantFlow: map[string][]string{
	// 		"x": {"x"}, "y": {"y"},
	// 		"t0": {"x"}, "t1": {"x"}, "t2": {"y"}, "t3": {"y"},
	// 	},
	// 	wantBlocks: [][]string{{"x"}, {"y"}},
	// },
	// {
	// 	// the pairs (x, y) and (x1, y1) never influence each other
	// 	funcName: "Pairs",
	// 	wantFlow: map[string][]string{
	// 		"x": {"x"}, "y": {"y"}, "x1": {"x1"}, "y1": {"y1"},
	// 		"t0": {"x"}, "t1": {"x", "y"}, "t2": {"x", "y"},
	// 		"t3": {"x1"}, "t4": {"x1", "y1"}, "t5": {"x1", "y1"},
	// 		"t6": {"x"}, "t7": {"x", "y"}, "t8": {"x", "y"},
	// 		"t9": {"x1"}, "t10": {"x1", "y1"}, "t11": {"x1", "y1"},
	// 	},
	// 	wantBlocks: [][]string{{"x", "y"}, {"x1", "y1"}},
	// },
	// {
	// 	funcName: "TwoAssignmentPointsToSet",
	// 	wantFlow: map[string][]string{
	// 		"t0": {"x"}, "t1": {"y"},
	// 		"t2": {"y"},
	// 		"x":  {"x"}, "y": {"y"},
	// 	},
	// 	wantBlocks: [][]string{{"x", "y"}},
	// },
	// {
	// 	funcName: "TwoAssignmentWithIfPointsToSet",
	// 	wantFlow: map[string][]string{
	// 		"t0": {"x"}, "t1": {"y"},
	// 		"t2": {"x"}, "t3": {"x"},
	// 		"t4": {"x", "y"},
	// 		"x":  {"x"}, "y": {"y"},
	// 	},
	// 	wantBlocks: [][]string{{"x", "y"}},
	// },
	// {
	// 	funcName: "OneAssignmentPointsToSet",
	// 	wantFlow: map[string][]string{
	// 		"t0": {"x", "y"}, "t1": {"y"},
	// 		"t2": {"x", "y"},
	// 		"x":  {"x"}, "y": {"y"},
	// 	},
	// 	wantBlocks: [][]string{{"x", "y"}},
	// },
	{
		funcName: "IntPointers",
		wantFlow: map[string][]string{
			"t0": {"x"}, "t1": {"y"},
			"t2": {"x"}, "t3": {"x"},
			"t4": {"x", "y"},
			"t5": {"x", "y"},
			"t6": {"x"},
			"t7": {"x", "y"},
			"t8": {"y"},
			"t9": {"x", "y"},
			"x":  {"x"}, "y": {"y"},
		},
		wantBlocks: [][]string{{"x", "y"}},
	},
	// {
	// 	funcName: "CreateStruct",
	// 	wantFlow: map[string][]string{
	// 		"t0.a": {"a"},
	// 		"t0.b": {"b"},
	// 		"t3.a": {"a"},
	// 		"t3.b": {"b"},
	// 		"a":    {"a"}, "b": {"b"},
	// 	},
	// },
	// {
	// 	funcName: "TestStructAssignment",
	// 	wantFlow: map[string][]string{
	// 		"t0.a":   {"aVal"},
	// 		"t0.b":   {"bVal"},
	// 		"t1.e.a": {"aVal"},
	// 		"t1.e.b": {"bVal"},
	// 		"t1.c":   {},
	// 		"t4.a":   {"aVal"},
	// 		"t4.b":   {"bVal"},
	// 		"aVal": {"aVal"}, "bVal": {"bVal"},
	// 	},
	// },
	// {
	// 	funcName: "StructArguments",
	// 	wantFlow: map[string][]string{
	// 		"t0.a": {"l.a"},
	// 		"t0.b": {"l.b"},
	// 		"t1.a": {"r.a"},
	// 		"t1.b": {"r.b"},
	// 		"t3":   {"l.a"},
	// 		"t5":   {"r.a"},
	// 		"t6":   {"l.a", "r.a"},
	// 		"t7":   {"l.a", "r.a"},
	// 		"t9":   {"l.b"},
	// 		"t11":  {"r.b"},
	// 		"t12":  {"l.b", "r.b"},
	// 		"t13":  {"l.b", "r.b"},
	// 		"l.a":  {"l.a"}, "l.b": {"l.b"},
	// 		"r.a": {"r.a"}, "r.b": {"r.b"},
	// 	},
	// },
	// {
	// 	funcName: "UseStructs",
	// 	wantFlow: map[string][]string{
	// 		"t0.Nested.String":   {"left.Nested.String"},
	// 		"t0.Nested.Int":      {"left.Nested.Int"},
	// 		"t0.AnotherInt":      {"left.AnotherInt"},
	// 		"t0.Bool":            {"left.Bool"},
	// 		"t1.Nested.String":   {"right.Nested.String"},
	// 		"t1.Nested.Int":      {"right.Nested.Int"},
	// 		"t1.AnotherInt":      {"right.AnotherInt"},
	// 		"t1.Bool":            {"right.Bool"},
	// 		"t6":                 {"left.Nested.Int"},
	// 		"t7":                 {"left.Nested.Int"},
	// 		"t10":                {"left.Nested.Int"},
	// 		"t11":                {"left.Nested.Int"},
	// 		"t13":                {"left.Nested.Int", "right.Bool"},
	// 		"t15":                {"left.Nested.Int", "right.Bool"},
	// 		"t16":                {"left.Nested.Int", "right.Bool"},
	// 		"t19":                {"left.Nested.String"},
	// 		"t20":                {"left.Nested.String"},
	// 		"t22":                {"left.Nested.String", "right.AnotherInt"},
	// 		"t23":                {"left.Nested.String", "right.AnotherInt"},
	// 		"t24":                {"left.Nested.String", "right.AnotherInt"},
	// 		"left.Nested.String": {"left.Nested.String"}, "left.Nested.Int": {"left.Nested.Int"},
	// 		"left.AnotherInt": {"left.AnotherInt"}, "left.Bool": {"left.Bool"},
	// 		"right.Nested.String": {"right.Nested.String"}, "right.Nested.Int": {"right.Nested.Int"},
	// 		"right.AnotherInt": {"right.AnotherInt"}, "right.Bool": {"right.Bool"},
	// 	},
	// },
}

func TestFindFlow(t *testing.T) {
	for _, tt := range flowTestCases {
		t.Run(tt.funcName, func(t *testing.T) {
			analyzer, err := NewAnalyzer(
				"./testdata/branching",
				tt.funcName,
				WithDumpSSAToFile("./testFileSSA.txt"),
				WithLogger(zaptest.NewLogger(t).Sugar()),
			)
			require.NoError(t, err)

			got := inputsOfFlow(analyzer.analyzedFunc.Params, analyzer.FindFlow())

			require.Equal(t, tt.wantFlow, got)
		})
	}
}

func TestExtractBlocks(t *testing.T) {
	for _, tt := range flowTestCases {
		t.Run(tt.funcName, func(t *testing.T) {
			analyzer, err := NewAnalyzer("./testdata/branching", tt.funcName)
			require.NoError(t, err)

			params := analyzer.analyzedFunc.Params
			blocks := analyzer.ExtractBlocks(params, analyzer.FindFlow())

			got := make([][]string, 0, len(blocks))
			for _, block := range blocks {
				names := make([]string, 0, len(block))
				for _, param := range block {
					names = append(names, param.Name())
				}
				got = append(got, names)
			}
			require.Equal(t, tt.wantBlocks, got)
		})
	}
}

// inputsOfFlow keeps only the function parameters in every flow set and prints them
// by name, so the result can be compared with a hand written expectation.
func inputsOfFlow(params []*ssa.Parameter, flow map[ssa.Value]*ssaValueSet) map[string][]string {
	isParam := getSSAValuesByParams(params)

	result := make(map[string][]string, len(flow))
	for val, set := range flow {
		names := []string{}
		for influencer := range set.All() {
			if isParam[influencer] {
				names = append(names, influencer.Name())
			}
		}
		slices.Sort(names)
		result[val.Name()] = names
	}

	return result
}

// getSSAValuesByParams returns the set of flow values that represent params.
// A struct param is tracked as one variable per leaf field (l.a, l.b, ...),
// so those field variables are included instead of just the param itself.
func getSSAValuesByParams(params []*ssa.Parameter) map[ssa.Value]bool {
	values := make(map[ssa.Value]bool, len(params))
	for _, param := range params {
		isStruct, structType := isStructType(param.Type())
		if !isStruct {
			values[param] = true
			continue
		}

		walkStructFields(structType, structFieldVariable{Value: param}, func(cell structFieldVariable) {
			values[cell] = true
		})
	}

	return values
}
