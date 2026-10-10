package d2mapengine

import "testing"

// campWorld is an open 300x300 sub-tile field (like the Rogue Encampment's
// walkable ground) with a closed 20x20 building in the middle.
func campWorld(x, y int) bool {
	if x < 0 || y < 0 || x >= 300 || y >= 300 {
		return true
	}

	return x >= 140 && x < 160 && y >= 140 && y < 160
}

// A click on a building: the goal is blocked and unreachable, so the search
// explores the whole bounded region before it settles for the nearest cell.
func BenchmarkFindPathBlockedGoal(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		findPath(campWorld, pt{20, 20}, pt{150, 150})
	}
}

// A click behind the building: the route bends around it.
func BenchmarkFindPathAroundBuilding(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		findPath(campWorld, pt{150, 100}, pt{150, 200})
	}
}

// The map based search, for comparison with findPath (dense).
func BenchmarkFindPathSparseBlockedGoal(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		findPathSparse(campWorld, pt{20, 20}, pt{150, 150})
	}
}
