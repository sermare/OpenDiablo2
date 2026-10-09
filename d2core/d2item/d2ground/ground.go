package d2ground

import (
	"fmt"
	"sort"
)

// Cell is a whole map tile coordinate.
type Cell struct{ X, Y int }

// DropCells returns up to n free cells for items dropped around (cx, cy),
// nearest first. Cells are visited in rings of growing Chebyshev radius; inside
// a ring the order is fixed (sorted by squared distance, then Y, then X) so the
// same inputs always give the same layout. The centre cell is used first, like
// the original, which looks for a free ground cell next to the dropper
// (FUN_00553e90 in Game.exe 1.14b; its exact search order is not recorded in
// the notes, so the ring order is UNVERIFIED). blocked reports cells that are
// not usable (not walkable, or already holding an item).
func DropCells(cx, cy, n, maxRadius int, blocked func(x, y int) bool) []Cell {
	var out []Cell

	for r := 0; r <= maxRadius && len(out) < n; r++ {
		var ring []Cell

		for y := cy - r; y <= cy+r; y++ {
			for x := cx - r; x <= cx+r; x++ {
				if maxInt(absInt(x-cx), absInt(y-cy)) != r || blocked(x, y) {
					continue
				}

				ring = append(ring, Cell{x, y})
			}
		}

		sort.Slice(ring, func(i, j int) bool {
			a, b := ring[i], ring[j]
			da, db := sq(a.X-cx)+sq(a.Y-cy), sq(b.X-cx)+sq(b.Y-cy)

			switch {
			case da != db:
				return da < db
			case a.Y != b.Y:
				return a.Y < b.Y
			default:
				return a.X < b.X
			}
		})

		for _, c := range ring {
			if len(out) == n {
				break
			}

			out = append(out, c)
		}
	}

	return out
}

func sq(a int) int { return a * a }

func absInt(a int) int {
	if a < 0 {
		return -a
	}

	return a
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}

	return b
}

// Difficulty selects the suffix of the chest treasure class names.
type Difficulty int

// Difficulties.
const (
	Normal Difficulty = iota
	Nightmare
	Hell
)

// ChestKinds are the letters of the "Act N Chest X" classes used here.
// ITEMGEN_BuildChestTreasureClassLookup (0x65c580) caches "Act %d%s Chest %s"
// for acts 1..5, the suffixes "", " (N)", " (H)" and four kinds; TreasureClassEx
// of 1.14b only contains A, B and C, which is what is modelled.
var ChestKinds = []string{"A", "B", "C"}

// ChestTreasureClass picks the treasure class of a chest: "Act N Chest X" with
// the difficulty suffix. ITEMGEN_DropChestTreasure (0x583a60) chooses the kind
// "by area level thirds" (notes); the real thresholds are not recorded, so this
// uses the Level column of the three table rows: the highest kind whose Level
// does not exceed the area level (UNVERIFIED), kind A below all of them.
// levelOf returns the Level column of a treasure class and whether it exists.
// It returns "" when no class of that act exists.
func ChestTreasureClass(act int, diff Difficulty, areaLevel int, levelOf func(name string) (int, bool)) string {
	suffix := ""

	switch diff {
	case Nightmare:
		suffix = " (N)"
	case Hell:
		suffix = " (H)"
	}

	if act < 1 {
		act = 1
	}

	best := ""

	for _, k := range ChestKinds {
		name := fmt.Sprintf("Act %d%s Chest %s", act, suffix, k)

		lvl, ok := levelOf(name)
		if !ok {
			continue
		}

		if best == "" || lvl <= areaLevel {
			best = name
		}
	}

	return best
}

// GoldAmount converts a rolled base amount into the gold of a pile. The
// treasure class entry "gld,mul=N" stores N+1 (notes: "mul is stored in +0xa as
// value+1") and the amount becomes (base*(N+1))>>8 (notes: "(stat 0xe * mul+1)
// >> 8"); entries without mul keep the base. Gold find then scales it by
// (100+goldFindPercent)/100 (ITEMGEN_ApplyGoldFindBonus, 0x556a10). The rule for
// the base itself is not in the notes; see BaseGold. The result is at least 1.
func GoldAmount(base, mul, goldFindPercent int) int {
	amount := base
	if mul > 0 {
		amount = (base * (mul + 1)) >> 8
	}

	amount = amount * (100 + goldFindPercent) / 100
	if amount < 1 {
		amount = 1
	}

	return amount
}

// BaseGold maps a roll in [0, 10*ilvl+10) to the base gold amount. The real
// formula of ITEMGEN_InitItemBaseStats is UNVERIFIED (the notes only say the
// amount starts from stat 0xe); this scales with the item level so deeper
// drops are richer, which is what the game does.
func BaseGold(ilvl int, roll func(n int32) uint32) int {
	if ilvl < 1 {
		ilvl = 1
	}

	return 1 + int(roll(int32(10*ilvl+10)))
}
