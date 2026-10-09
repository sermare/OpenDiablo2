package d2monreg

// The unique / champion branch of the natural population (FUN_005bba00 in
// mode 1, FUN_005a1ec0 = FUN_0059e570 + FUN_0059e330, FUN_0054c050).
//
// What is exact: the class draw, the spot search and the placement (room
// seed), the unit's own seed, the champion decision (unit seed, 20 percent:
// monumod.txt constants row), the number and the placement of the champion
// minions, the unique counter of the region.
//
// What is a model: the modifier pick of FUN_0059e1e0 / FUN_0059e0e0 draws one
// number from the unit seed per modifier over the weights of the eligible
// monumod rows; the eligibility test (FUN_0059dfc0) is not modelled, so every
// pick is assumed to find at least one candidate. That only matters for the
// unit's own seed, i.e. for how many minions a champion gets.

// ChampionChance is monumod.txt row 0, constants column (the "champion
// chance" percentage).
const ChampionChance = 20

const maxUnitMods = 9

// assignMods is FUN_0059e330 with its flag set (natural unique packs).
func (g *Game) assignMods(u *Unit) (champion bool) {
	if int(u.Seed.Roll(100)) < ChampionChance {
		u.Champion = true
		u.Seed.Roll(2) // FUN_0059e0e0: one pick over the champion mods (the weight total is not modelled; any total > 0 takes one step)

		return true
	}

	n := int(u.Seed.Roll(1)) + 1 + g.Difficulty
	if n > maxUnitMods-1 {
		n = maxUnitMods - 1
	}

	for ; n > 0; n-- {
		u.Seed.Roll(2) // FUN_0059e1e0: one pick over the unique mods
	}

	return false
}

// uniquePack creates one unique or champion pack leader in the cell.
func (g *Game) uniquePack(w World, room *Room, r *Region, c *Cell, pop *Population) bool {
	class, ok := g.pickType(r, room, 0, true)
	if !ok {
		return false
	}

	// FUN_0059e570 with no position: FUN_0054ba70 finds one (exit test on)
	x, y, ok := g.pickPos(w, room, c, class, true)
	if !ok {
		return false
	}

	px, py, ok := g.Tables.Place(w, Placement{Room: room, Cell: c, Class: class, X: x, Y: y, Ring: -1, Flags: 0x40, Param4: 1})
	if !ok {
		return false
	}

	u := g.create(pop, room, class, px, py)
	u.Unique = true
	r.Uniques++ // FUN_0059df00

	g.assignMods(u)

	if u.Champion {
		pop.Log = append(pop.Log, Event{Kind: "champ", Class: class})
	} else {
		pop.Log = append(pop.Log, Event{Kind: "uniq", Class: class})
	}

	if !u.Champion {
		return true
	}

	// FUN_0054c050: 1..3 champion minions on the ring of radius 4
	n := int(u.Seed.Roll(3)) + 1

	for ; n > 0; n-- {
		if m := g.follower(w, room, c, u, class, 4, pop); m != nil {
			m.Champion = true
		}
	}

	return true
}
