package d2maprenderer

import "fmt"

// DrawStats counts what one Render call drew. The game logs it (DRAWSTATS) when OD2_DRAWSTATS is set, so a
// screenshot scenario can assert that a scene was really drawn (floors, walls, shadows, lit entities...).
type DrawStats struct {
	Floors, Walls, UpperWalls, Shadows, Roofs int
	RoofsFading, WallsFading                  int // roofs / upper walls with alpha < 1
	Entities                                  int
	LightSources                              int // hero + object lights feeding the light map
	Lit                                       bool
}

// String is the log form.
func (s DrawStats) String() string {
	return fmt.Sprintf("floors=%d walls=%d upperwalls=%d shadows=%d roofs=%d roofsfading=%d wallsfading=%d entities=%d lightsources=%d lit=%v",
		s.Floors, s.Walls, s.UpperWalls, s.Shadows, s.Roofs, s.RoofsFading, s.WallsFading, s.Entities, s.LightSources, s.Lit)
}

// DrawStats returns the statistics of the last Render call.
func (mr *MapRenderer) DrawStats() DrawStats { return mr.stats }
