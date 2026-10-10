package d2monster

// GargoyleTrap facing (MONAI_Think_GargoyleTrap 0x5f8600, VERIFIED in
// ai-faithful-batch4.md follow-ups). Before acting the statue turns toward its
// target along the dominant cardinal axis: the direction is taken (0x621f80)
// toward the point (own x, target y) when |dx| < |dy|, else toward (target x,
// own y). The 64-way direction d maps to the quadrant ((d + 7) >> 4) & 3 and
// the facing byte is table 0x6e4acc[quadrant].

// gargoyleFacing is the byte table at 0x6e4acc.
var gargoyleFacing = [4]int{0x1f, 0x31, 0x00, 0x11}

// GargoyleFacePoint is the point the statue looks at for a target at (tx, ty).
func GargoyleFacePoint(ownX, ownY, tx, ty int) (x, y int) {
	dx, dy := tx-ownX, ty-ownY
	if dx < 0 {
		dx = -dx
	}

	if dy < 0 {
		dy = -dy
	}

	if dx < dy {
		return ownX, ty
	}

	return tx, ownY
}

// GargoyleFacingByte maps a 64-way direction to the facing byte.
func GargoyleFacingByte(dir64 int) int {
	return gargoyleFacing[((dir64+7)>>4)&3]
}

// Facer is the optional host hook that turns the unit. x, y is the point it
// looks at (GargoyleFacePoint); facing is the exe's facing byte, or -1 when
// the host cannot compute the 64-way direction (no Direction64Mapper).
type Facer interface {
	FaceTowards(b *Brain, x, y, facing int)
}

// Direction64Mapper is UNIT_GetDirectionToPoint (0x621f80): the 64-way
// direction from the unit to a point (orientation of direction 0 UNVERIFIED).
type Direction64Mapper interface {
	Direction64(b *Brain, x, y int) int
}

func (c *Ctx) gargoyleFace(t Target) {
	f, ok := c.W.(Facer)
	if !ok {
		return
	}

	x, y := GargoyleFacePoint(c.B.X, c.B.Y, t.X, t.Y)
	facing := -1

	if m, ok := c.W.(Direction64Mapper); ok {
		facing = GargoyleFacingByte(m.Direction64(c.B, x, y))
	}

	f.FaceTowards(c.B, x, y, facing)
}
