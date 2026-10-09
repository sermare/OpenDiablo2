package d2mapengine

// ObjectCollision is a mutable overlay of blocked sub-tiles for objects whose
// collision changes at run time (doors). The DT1 BlockWalk flags are baked
// into the tiles; the original keeps a mutable flag word per sub-tile in its
// collision grid (missiles-pathing.md: bit 0x800 = closed door). This overlay
// is the small part of that grid the engine needs for doors: each owner id
// stamps a rectangle, and a sub-tile is blocked while any rectangle covers it.
type ObjectCollision struct {
	owners map[string]rect
	count  map[[2]int]int
}

type rect struct{ x, y, w, h int }

// NewObjectCollision returns an empty overlay.
func NewObjectCollision() *ObjectCollision {
	return &ObjectCollision{owners: map[string]rect{}, count: map[[2]int]int{}}
}

// Set blocks the sub-tile rectangle for owner, replacing the owner's old one.
func (c *ObjectCollision) Set(owner string, x, y, w, h int) {
	c.Clear(owner)

	c.owners[owner] = rect{x, y, w, h}

	for sy := y; sy < y+h; sy++ {
		for sx := x; sx < x+w; sx++ {
			c.count[[2]int{sx, sy}]++
		}
	}
}

// Clear removes the owner's rectangle (no-op if it has none).
func (c *ObjectCollision) Clear(owner string) {
	r, ok := c.owners[owner]
	if !ok {
		return
	}

	delete(c.owners, owner)

	for sy := r.y; sy < r.y+r.h; sy++ {
		for sx := r.x; sx < r.x+r.w; sx++ {
			k := [2]int{sx, sy}
			if c.count[k]--; c.count[k] <= 0 {
				delete(c.count, k)
			}
		}
	}
}

// Blocked reports whether a sub-tile is covered by any rectangle.
func (c *ObjectCollision) Blocked(x, y int) bool {
	return c.count[[2]int{x, y}] > 0
}

// Owners returns the number of owners that currently block something.
func (c *ObjectCollision) Owners() int { return len(c.owners) }
