package branching

func Pairs(x, y, x1, y1 int) (int, int) {
	if x != 0 {
		y++
	}
	if x1 != 0 {
		y1++
	}

	a, b := 0, 0
	if x != 0 {
		if y == 1 {
			a = 1
		}
	}
	if x1 != 0 {
		if y1 == 1 {
			b = 2
		}
	}
	return a, b
}
