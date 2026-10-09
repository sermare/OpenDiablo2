package d2interface

import "image/color"

// ShadedSurface is implemented by surfaces that can draw another surface with
// a per-vertex colour (gouraud shading), which the map renderer uses for the
// light map: the source is split into cols x rows quads and shade is called
// for each (cols+1)*(rows+1) vertex with its pixel position inside the source.
type ShadedSurface interface {
	RenderShaded(src Surface, cols, rows int, shade func(x, y float64) color.RGBA)
}
