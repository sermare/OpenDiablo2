package d2mapentity

import "image/color"

// StateTinted holds the colour a unit's states give it (states.txt colorshift: frozen blue, poisoned green...).
// The original remaps the unit's palette through a PL2 hue row (FUN_004d65a0 picks the state, unit +0x6c holds the
// shift); the RGBA renderer approximates the row as a multiplier (d2pl2.HueTint). The renderer multiplies it into the
// light tint of the unit (d2maprenderer.renderLitEntity).
type StateTinted struct {
	tintR, tintG, tintB float64
	tintOn              bool
}

// SetStateTint sets the multiplier; on false clears it.
func (s *StateTinted) SetStateTint(r, g, b float64, on bool) {
	s.tintR, s.tintG, s.tintB, s.tintOn = r, g, b, on
}

// StateTint returns the multiplier and whether a state colours the unit.
func (s *StateTinted) StateTint() (r, g, b float64, on bool) {
	return s.tintR, s.tintG, s.tintB, s.tintOn
}

// MulStateTint multiplies a light tint by a state tint.
func MulStateTint(c color.RGBA, r, g, b float64) color.RGBA {
	return color.RGBA{R: uint8(float64(c.R) * r), G: uint8(float64(c.G) * g), B: uint8(float64(c.B) * b), A: c.A}
}
