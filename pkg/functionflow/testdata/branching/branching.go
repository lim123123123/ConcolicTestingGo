package branching

func Branching(x, y int) int {
	z := 0
	if x == 1 {
		y = y + 3
		if y == 4 {
			z = 2
		}
	}
	return z + x + y
}
