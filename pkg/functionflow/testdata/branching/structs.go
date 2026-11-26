package branching

type SimpleStructWithTwoFields struct {
	a int
	b int
}

// Test that alloc creates the right keys in flow
func CreateStruct(a, b int) SimpleStructWithTwoFields {
	simpleStruct := SimpleStructWithTwoFields{}
	simpleStruct.a = a
	simpleStruct.b = b

	return simpleStruct
}

type EmbeddedStruct struct {
	a int
	b int
}

type OuterStruct struct {
	e EmbeddedStruct
	c int
}

// Test struct = struct assignment
func TestStructAssignment(aVal, bVal int) {
	a := EmbeddedStruct{}
	b := OuterStruct{}
	a.a = aVal
	a.b = bVal
	b.e = a
}

func StructArguments(l, r SimpleStructWithTwoFields) (int, int) {
	var a, b int
	if l.a == r.a {
		a = 1
	}

	if l.b == r.b {
		b = 1
	}

	return a, b
}

type MyStructTypeNested struct {
	String string
	Int    int
}

type MyStruct struct {
	Nested     MyStructTypeNested
	AnotherInt int
	Bool       bool
}

func UseStructs(left, right MyStruct) (int, int) {
	a := 0
	b := 0
	left.Nested.Int = 3
	if left.Nested.Int == 1 && right.Bool {}
	if left.Nested.Int == 1 {
		if right.Bool {
			a = 1
		}
	}

	if left.Nested.String == "42" {
		if right.AnotherInt == 1 {
			b = 1
		}
	}
	return a, b
}
