package d2interface

import "image/color"

// ShadedSurface is implemented by surfaces that can draw another surface with
// a per-vertex colour (gouraud shading), which the map renderer uses for the
// light map: the source is split into cols x rows quads and shade is called
// for each (cols+1)*(rows+1) vertex with its pixel position inside the source.
type ShadedSurface interface {
	RenderShaded(src Surface, cols, rows int, shade func(x, y float64) color.RGBA)
}

// ShadedGridSurface is the allocation-free form of ShadedSurface: vals holds the (cols+1)*(rows+1)
// vertex colours row by row (the values RenderShaded's callback would return for the grid vertices),
// and the surface does not keep vals after the call.
type ShadedGridSurface interface {
	RenderShadedGrid(src Surface, cols, rows int, vals []color.RGBA)
}
