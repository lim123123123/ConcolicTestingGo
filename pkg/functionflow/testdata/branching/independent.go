package branching

func Independent(x int, y int) (int, int) {
	a, b := 0, 0
	if x == 30 {
		a = 1
	}
	if y == 30 {
		b = 2
	}
	return a, b
}
