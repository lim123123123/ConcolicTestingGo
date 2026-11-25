package branching

func TwoAssignmentPointsToSet(x, y int) int {
	var ptr *int
	ptr = &x
	ptr = &y

	// Flow of ptr must contain both x and y
	return *ptr
}

func TwoAssignmentWithIfPointsToSet(x, y int) *int {
	var ptr *int

	if x == 0 {
		ptr = &x
	} else {
		ptr = &y
	}

	// Flow of ptr must contain both x and y
	return ptr
}

func OneAssignmentPointsToSet(x, y int) int {
	x = y + 5
	var ptr *int
	ptr = &x

	// Flow of ptr must contain both x and y
	return *ptr
}

// Test that we actually over approximate the
// points-to set
func IntPointers(x, y int) int {
	var xPtr *int
	if x == 0 {
		xPtr = &y
		x = 2
	} else {
		xPtr = &x
	}

	// flow for xPtr must contain both and y
	return *xPtr + x + y
}
