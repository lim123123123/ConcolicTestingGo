package branching

func BreakFlowTest(x, z int) {
	c := 0
	if x != 0 {
		c = 2
	}

	if c == 2 {
		if z != 0 {
			panic("bad situation")
		}
	}
}
