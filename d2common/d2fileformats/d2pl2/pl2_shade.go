package d2pl2

// ShadeRows is the number of light levels the game indexes (rows 0..31 of the
// 49 shade rows; 31 is the identity = full bright). Rows 32..48 are brighter
// or colour variants and are not reached by light intensities.
const ShadeRows = 32

// ShadeRow maps a light intensity byte (0..255) to its PL2 shade row. This is
// the software renderer's rule in GFX_DrawCelSoftware (rowtable[light >> 3]),
// verified in d2-re-notes/renderer.md.
func ShadeRow(intensity int) int {
	if intensity < 0 {
		return 0
	}

	if intensity > 0xff {
		intensity = 0xff
	}

	return intensity >> 3 //nolint:gomnd // 256 -> 32 rows
}

// LinearShadeFactors is the fallback when no PL2 is available: row/31.
func LinearShadeFactors() [ShadeRows]float64 {
	var f [ShadeRows]float64
	for i := range f {
		f[i] = float64(i) / float64(ShadeRows-1)
	}

	return f
}

func luminance(c PL2Color) float64 {
	return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B) //nolint:gomnd // Rec.601
}

// ShadeFactors measures each light row as a brightness multiplier relative to
// the base palette: the summed luminance of the palette after the row's
// remap, divided by that of the unmapped palette, clamped to 0..1. This lets
// an RGBA renderer approximate the paletted shading by multiplying colours.
// The PL2 rows darken toward black non-linearly and keep some tint, so this
// is an approximation (see the notes, section c.6).
func (p *PL2) ShadeFactors() [ShadeRows]float64 {
	var base float64

	for _, c := range p.BasePalette.Colors {
		base += luminance(c)
	}

	var f [ShadeRows]float64
	if base == 0 {
		return LinearShadeFactors()
	}

	for r := 0; r < ShadeRows; r++ {
		var sum float64

		for _, idx := range p.LightLevelVariations[r].Indices {
			sum += luminance(p.BasePalette.Colors[idx])
		}

		f[r] = sum / base
		if f[r] > 1 {
			f[r] = 1
		}
	}

	return f
}
