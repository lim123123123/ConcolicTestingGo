package functionflow

import (
	"go/token"
	"go/types"

	"golang.org/x/tools/go/ssa"
)

// We represent structs as a union of variables, so we could find dependencies between different struct fields
// see readme for more details. This struct is a wrapper that represents a variable for a struct field leaf
type structFieldVariable struct {
	ssa.Value            // The original variable for the struct itself
	fieldName string     // dot separated path to the field that current structFieldVariable represent
	fieldType types.Type // basic type of a field that this structFieldVariable wraps
}

func (cell structFieldVariable) Name() string     { return cell.Value.Name() + "." + cell.fieldName }
func (cell structFieldVariable) String() string   { return cell.Name() }
func (cell structFieldVariable) Type() types.Type { return cell.fieldType }

func (cell structFieldVariable) AddField(field *types.Var) structFieldVariable {
	return structFieldVariable{
		Value:     cell.Value,
		fieldName: joinWithDot(cell.fieldName, field.Name()),
		fieldType: field.Type().Underlying(),
	}
}

// Other represents a subpath from cell, useful when you are only calculating the leafs
func (cell structFieldVariable) Merge(other structFieldVariable) structFieldVariable {
	return structFieldVariable{
		Value:     cell.Value,
		fieldName: joinWithDot(cell.fieldName, other.fieldName),
		fieldType: other.fieldType,
	}
}

func joinWithDot(prefix, suffix string) string {
	if prefix == "" {
		return suffix
	}
	return prefix + "." + suffix
}

// Returns type to which pointer points
func getPtrType(ptr ssa.Value) types.Type {
	return ptr.Type().Underlying().(*types.Pointer).Elem().Underlying()
}

func isStructType(t types.Type) (bool, *types.Struct) {
	// https://github.com/golang/example/tree/master/gotypes#types
	switch underlyingStruct := t.Underlying().(type) {
	case *types.Named:
		panic("Still have named type after Underlying() call")
	case *types.Struct:
		return true, underlyingStruct
	default:
		return false, nil
	}
}

// In alloc we must add a union of variables in case if the allocated variable
// is a struct and just one variable in case of a basic type
func (analyzer *FunctionFlowAnalyzer) handleAllocInstr(
	alloc *ssa.Alloc,
	ctrl *ssaValueSet,
) {
	isStruct, structType := isStructType(getPtrType(alloc))
	if !isStruct {
		// Allocating a basic type, just add a new block to the flow
		analyzer.getOrCreateFlow(alloc)

		for dep := range ctrl.All() {
			analyzer.getOrCreateFlow(alloc).Merge(analyzer.getOrCreateFlow(dep))
		}

		return
	}

	// We are allocating a struct, recursively walk all the fields and add them to the flow
	walkStructFields(structType, structFieldVariable{
		Value: alloc,
	}, func(sfv structFieldVariable) {
		analyzer.getOrCreateFlow(sfv)
	})
}

// There are 3 cases:
// The first one is when the store happens to a location that was initially allocated as a basic type
// that means that we are trying to store sth in a pointer to a basic type
// The second one is when the pointer is pointing to a struct field, and was obtained by FieldAddr
// and the field itself is a basic type. In this case we just need to resolve the initial name
// The third one is when the field is a struct, in this case we must recursively walk all the leaves of the struct
// *x = t
func (analyzer *FunctionFlowAnalyzer) handleStoreInstr(
	store *ssa.Store,
	ctrl *ssaValueSet,
) {
	// If we are assigning a const there are no data dependencies
	if _, ok := store.Val.(*ssa.Const); ok {
		return
	}

	// Find the var to which the pointer may point
	pointerVar := findPointsToForValue(store.Addr, "")
	switch typedPointerVar := pointerVar.fieldType.(type) {
	case *types.Basic:
		// The pointer points to a basic type variable, and on the right
		// we have a standard variable as well, so just merge flow
		analyzer.getOrCreateFlow(pointerVar).Merge(
			analyzer.getOrCreateFlow(store.Val),
		)
		analyzer.mergeCtrlDependencies(pointerVar, ctrl)
	case *types.Struct:
		// The pointer points to a struct, so right side is a struct as well
		// we need to walk all the variables related to the struct fields
		analyzer.assignOneStructToAnother(typedPointerVar, pointerVar, structFieldVariable{
			Value: store.Val,
		}, ctrl)
	default:
		panic("unknown ptr type in handleStoreInstr")
	}
}

// Returns the variable to which ptr points to and a path to it
// a path is non-empty if the ptr points to a struct field
// NOTE: see readme for mode details, of course it should be a normal
// points to relation, but it's enough for our analysis to just follow all FieldAddr instructions
func findPointsToForValue(ptr ssa.Value, path string) structFieldVariable {
	switch tInstr := ptr.(type) {
	case *ssa.Alloc:
		// If we reach alloc we could find the original struct variable (the value of alloc)
		return structFieldVariable{
			Value:     tInstr,
			fieldName: "",
			fieldType: getPtrType(tInstr),
		}
	case *ssa.FieldAddr:
		field := getPtrType(tInstr.X).(*types.Struct).Field(tInstr.Field)
		fieldVar := findPointsToForValue(tInstr.X, path)

		return fieldVar.AddField(field)
	default:
		panic("unknown instruction in build canonical name")
	}
}

// Calculate flow for assignment of two structs *left = right
// the structs are represented as structFieldCell as they could be
// a subfields of some other structs
func (analyzer *FunctionFlowAnalyzer) assignOneStructToAnother(
	structType *types.Struct,
	left structFieldVariable,
	right structFieldVariable,
	ctrl *ssaValueSet,
) {
	walkStructFields(structType, structFieldVariable{}, func(cell structFieldVariable) {
		// Here we just have a path that was gently build for us
		flowOfLeft := analyzer.getOrCreateFlow(left.Merge(cell))
		flowOfRight := analyzer.getOrCreateFlow(right.Merge(cell))
		flowOfLeft.Merge(flowOfRight)
		analyzer.mergeCtrlDependencies(left.Merge(cell), ctrl)
	})
}

// TODO: what if the structure has a pointer? What if this pointer is recursive
// Walk all the fields of the struct and attach them to the parent cell
func walkStructFields(
	structType *types.Struct,
	parent structFieldVariable,
	action func(cell structFieldVariable),
) {
	for field := range structType.Fields() {
		fieldType := field.Type()
		nextNode := parent.AddField(field)

		// Go down to the recursion
		if isStruct, fieldStructType := isStructType(fieldType); isStruct {
			walkStructFields(fieldStructType, nextNode, action)
			continue
		}

		action(nextNode)
	}
}

// Handle t = &x.field
// Now we just do nothing, as we try to statically resolve all the pointers through findPointsToForValue
// However, this function should update the over approximation of points-to relation, by adding a new
// pointer that points to a struct subfield to it
func (analyzer *FunctionFlowAnalyzer) handleFieldAddrInstr(faInstr *ssa.FieldAddr, ctrl *ssaValueSet) {
	// Empty on purpose
}

func (analyzer *FunctionFlowAnalyzer) handleUnOp(unOp *ssa.UnOp, ctrl *ssaValueSet) {
	// TODO: if we have a constant here, we should still add control dependencies
	if _, ok := unOp.X.(*ssa.Const); ok {
		// If we are assigning a constant - there are no data dependencies
		return
	}

	if unOp.Op != token.MUL {
		// This is not a load operation, so it's just an arithmetic operation
		analyzer.getOrCreateFlow(unOp).Merge(analyzer.getOrCreateFlow(unOp.X))
		analyzer.mergeCtrlDependencies(unOp, ctrl)
	} else {
		analyzer.handleLoadInstr(unOp, ctrl)
	}
}

// Handle t = *x operation
// Requires: loadInstr.Op == token.MUL
func (analyzer *FunctionFlowAnalyzer) handleLoadInstr(loadInstr *ssa.UnOp, ctrl *ssaValueSet) {
	if _, ok := loadInstr.X.Type().Underlying().(*types.Pointer); !ok {
		panic("not a pointer type in handleLoadInstr")
	}

	// Find all the values to which x may point to
	fieldVar := findPointsToForValue(loadInstr.X, "")

	switch fieldTypeTyped := fieldVar.fieldType.(type) {
	case *types.Basic:
		// Ptr points to a basic type
		analyzer.getOrCreateFlow(loadInstr).Merge(
			analyzer.getOrCreateFlow(fieldVar),
		)
		analyzer.mergeCtrlDependencies(loadInstr, ctrl)
	case *types.Struct:
		// We need to assign one struct to an another
		// t here is not a pointer, it's just a variable of type struct, so it's easy to create
		// corresponding struct field variables for it
		walkStructFields(fieldTypeTyped, fieldVar, func(xVar structFieldVariable) {
			tVar := xVar
			tVar.Value = loadInstr

			analyzer.getOrCreateFlow(tVar).Merge(
				analyzer.getOrCreateFlow(xVar),
			)
			analyzer.mergeCtrlDependencies(tVar, ctrl)
		})
	default:
		panic("unknown field type in handleLoadInstr")
	}
}
