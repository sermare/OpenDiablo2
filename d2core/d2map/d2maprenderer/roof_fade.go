package d2maprenderer

// Roofs hide while the hero is under cover (renderer.md b1 pass D: roof tiles are drawn last; which tiles get
// the fade flags is set in the room code, which the notes did not locate (U)). The rule used here: the roof
// tiles connected to the roof tile the hero stands on form one building and fade out together; all other
// roofs stay. The fade is linear over 500 ms like the wall fade.

const (
	// roofRegionMax caps the flood fill (a whole-map roof blob would mean bad data).
	roofRegionMax = 4096
	roofFadeTo    = 0.0
)

// roofRegion returns the set of tiles of the roof blob the tile (hx, hy) belongs to, using 8-connectivity over
// the tiles for which hasRoof is true. It returns nil when the hero's tile has no roof or the blob is bigger
// than roofRegionMax tiles.
func roofRegion(hasRoof func(x, y int) bool, hx, hy int) map[[2]int]bool {
	if !hasRoof(hx, hy) {
		return nil
	}

	seen := map[[2]int]bool{{hx, hy}: true}
	stack := [][2]int{{hx, hy}}

	for len(stack) > 0 {
		c := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				n := [2]int{c[0] + dx, c[1] + dy}
				if seen[n] || !hasRoof(n[0], n[1]) {
					continue
				}

				seen[n] = true
				if len(seen) > roofRegionMax {
					return nil
				}

				stack = append(stack, n)
			}
		}
	}

	return seen
}

// fadeAlpha moves a fade alpha one step toward target (1 = visible) over fadeSeconds of the full range.
func fadeAlpha(alpha, target, elapsed, fadeSeconds float64) float64 {
	step := elapsed / fadeSeconds

	switch {
	case alpha > target:
		alpha -= step
		if alpha < target {
			alpha = target
		}
	case alpha < target:
		alpha += step
		if alpha > target {
			alpha = target
		}
	}

	return alpha
}

// roofAlpha advances and returns the alpha of one roof tile.
func (l *lighting) roofAlpha(key wallKey, covering bool) float64 {
	f := l.fades[key]
	if f == nil {
		if !covering {
			return 1
		}

		f = &wallFade{alpha: 1}
		l.fades[key] = f
	}

	target := 1.0
	if covering {
		target = roofFadeTo
	}

	f.alpha = fadeAlpha(f.alpha, target, l.elapsed, wallFadeSeconds)
	if f.alpha >= 1 && !covering {
		delete(l.fades, key)
		return 1
	}

	return f.alpha
}
