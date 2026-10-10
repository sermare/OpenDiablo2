package d2mapengine

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2dt1"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2vector"
)

// WalkProbeResult counts the map tiles whose every visible floor is lava (or
// water) and how many of them a hero standing at the probe origin can reach
// by walking. In the original none can be walked on.
type WalkProbeResult struct {
	Lava, LavaReachable   int
	Water, WaterReachable int
	// Bridge counts tiles that have a lava/water floor layer below a walkable
	// floor layer (bridges) and BridgeReachable those a hero can reach.
	Bridge, BridgeReachable int
}

func (m *MapEngine) floorMaterial(c *MapTile) (lava, water, other int) {
	for i := range c.Components.Floors {
		f := &c.Components.Floors[i]
		if f.Hidden() || f.Prop1 == 0 {
			continue
		}

		opts := m.GetTiles(int(f.Style), int(f.Sequence), 0)
		if len(opts) == 0 {
			continue
		}

		idx := int(f.RandomIndex)
		if idx >= len(opts) {
			idx = 0
		}

		var mat d2dt1.MaterialFlags = opts[idx].MaterialFlags

		switch {
		case mat.Lava:
			lava++
		case mat.Water:
			water++
		default:
			other++
		}
	}

	return lava, water, other
}

// WalkProbe classifies the lava and water tiles of the map and checks which
// can be reached from the sub-tile (subX, subY) (the hero position). It uses
// ReachableFrom, so the answer is the one the click-to-move search gives.
func (m *MapEngine) WalkProbe(subX, subY int) WalkProbeResult {
	var res WalkProbeResult

	reach := m.ReachableFrom(subX, subY)

	for ty := 0; ty < m.size.Height; ty++ {
		for tx := 0; tx < m.size.Width; tx++ {
			t := &m.tiles[m.tileCoordinateToIndex(tx, ty)]

			lava, water, other := m.floorMaterial(t)
			if lava+water == 0 {
				continue
			}

			any := false

			for sy := 0; sy < subtilesPerTile && !any; sy++ {
				for sx := 0; sx < subtilesPerTile; sx++ {
					if reach.At(tx*subtilesPerTile+sx, ty*subtilesPerTile+sy) {
						any = true
						break
					}
				}
			}

			switch {
			case other > 0:
				res.Bridge++

				if any {
					res.BridgeReachable++
				}
			case lava > 0:
				res.Lava++

				if any {
					res.LavaReachable++
				}
			default:
				res.Water++

				if any {
					res.WaterReachable++
				}
			}
		}
	}

	return res
}

// IsLavaOrWaterOnly reports that every visible floor of the tile is lava or
// water.
func (m *MapEngine) IsLavaOrWaterOnly(tx, ty int) bool {
	if tx < 0 || ty < 0 || tx >= m.size.Width || ty >= m.size.Height {
		return false
	}

	lava, water, other := m.floorMaterial(&m.tiles[m.tileCoordinateToIndex(tx, ty)])

	return lava+water > 0 && other == 0
}

// TryWalkOntoLava orders the click-to-move search to the nearest lava/water
// tile (centre sub-tile) and reports that tile and whether the route it found
// ends on lava/water. The original never lets the hero stand there.
func (m *MapEngine) TryWalkOntoLava(subX, subY int) (tx, ty int, endsOnLava, found bool) {
	best := -1

	for y := 0; y < m.size.Height; y++ {
		for x := 0; x < m.size.Width; x++ {
			if !m.IsLavaOrWaterOnly(x, y) {
				continue
			}

			dx, dy := x*subtilesPerTile+2-subX, y*subtilesPerTile+2-subY
			d := dx*dx + dy*dy

			if best < 0 || d < best {
				best, tx, ty, found = d, x, y, true
			}
		}
	}

	if !found {
		return 0, 0, false, false
	}

	prev := m.gridPaths
	m.UseCollisionPaths(true)
	defer m.UseCollisionPaths(prev)

	route := m.PathFind(d2vector.NewPosition(float64(subX), float64(subY)),
		d2vector.NewPosition(float64(tx*subtilesPerTile+2), float64(ty*subtilesPerTile+2)))
	if len(route) == 0 {
		return tx, ty, false, true
	}

	end := route[len(route)-1]

	return tx, ty, m.IsLavaOrWaterOnly(int(end.X())/subtilesPerTile, int(end.Y())/subtilesPerTile), true
}
