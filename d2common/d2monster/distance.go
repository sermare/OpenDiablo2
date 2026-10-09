package d2monster

// Distance is the metric used by the AI (VERIFIED, 0x5db280): with
// dx=|x1-x2| and dy=|y1-y2| it is max(dx,dy)+min(dx,dy)/2 (integer division).
func Distance(dx, dy int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dx < dy {
		dx, dy = dy, dx
	}

	return dx + dy/2
}

// EdgeDistance is Distance after subtracting a unit size from each axis,
// clamped at zero (VERIFIED for the "first subtracts the unit's size" part of
// MONAI_GetEdgeDistanceBetweenUnits; which unit's size is subtracted, and how
// size is derived, is UNVERIFIED, so a single size is taken here).
func EdgeDistance(dx, dy, size int) int {
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	dx -= size
	dy -= size

	if dx < 0 {
		dx = 0
	}

	if dy < 0 {
		dy = 0
	}

	return Distance(dx, dy)
}
