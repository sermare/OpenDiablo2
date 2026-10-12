package d2pl2

// HueRows is the number of hue variation rows a state colour shift can pick
// (states.txt colorshift, 0..110, verified in d2state tests).
const HueRows = 111

// HueTint approximates a hue variation row as an RGB multiplier for a
// renderer that holds RGBA sprites instead of palette indices. The original
// recolours a unit by remapping every palette index through the row
// (states.txt colorshift -> HueVariations[shift]); here each channel is the
// luminance-weighted sum of the remapped palette over that of the plain
// palette, then scaled so the largest channel is 1 (the tint only shifts the
// hue, it must not brighten) and kept above 0.2 so no channel dies.
// ok is false when the shift is out of range or the palette is empty; the
// result is an approximation (UNVERIFIED which palette the exe maps through).
func (p *PL2) HueTint(shift int) (r, g, b float64, ok bool) {
	if p == nil || shift < 0 || shift >= HueRows {
		return 1, 1, 1, false
	}

	var base, mapped [3]float64

	row := p.HueVariations[shift].Indices

	for i := 1; i < 256; i++ { // index 0 is the transparent colour
		c := p.BasePalette.Colors[i]
		w := luminance(c)
		d := p.BasePalette.Colors[row[i]]

		base[0] += w * float64(c.R)
		base[1] += w * float64(c.G)
		base[2] += w * float64(c.B)
		mapped[0] += w * float64(d.R)
		mapped[1] += w * float64(d.G)
		mapped[2] += w * float64(d.B)
	}

	var t [3]float64

	maxv := 0.0

	for k := range t {
		if base[k] <= 0 {
			return 1, 1, 1, false
		}

		t[k] = mapped[k] / base[k]
		if t[k] > maxv {
			maxv = t[k]
		}
	}

	if maxv <= 0 {
		return 1, 1, 1, false
	}

	for k := range t {
		t[k] /= maxv
		if t[k] < 0.2 { //nolint:gomnd // floor
			t[k] = 0.2
		}
	}

	return t[0], t[1], t[2], true
}
