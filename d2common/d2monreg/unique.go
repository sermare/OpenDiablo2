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

// assignMods is FUN_0059e330 with its flag set (natural unique packs). With the
// monumod table loaded the picks are the exe's (weights, eligibility, the
// champion row); without it each pick only takes its one seed step.
func (g *Game) assignMods(u *Unit) (champion bool) {
	if ut := g.Tables.UMods; ut != nil {
		u.Mods, champion = ut.Roll(g.Tables, &u.Seed, g.Difficulty, u.Class, g.Expansion, true, u.Mods)
		u.Champion = champion

		return champion
	}

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

// minionClass is MONSTER_GetMinionBaseClassId: the class' minion1 when it is a
// valid class, else the class itself.
func (g *Game) minionClass(class int) int {
	if class >= 0 && class < len(g.Tables.Mons) {
		if m1 := g.Tables.Mons[class].Minion1; m1 >= 0 && m1 < len(g.Tables.Mons) {
			return m1
		}
	}

	return class
}

// link makes f a minion of leader and gives it the xfer modifiers of the
// leader (MONSTER_CopyLeaderUModsToMinion).
func (g *Game) link(leader, f *Unit) {
	f.Minion = true

	if ut := g.Tables.UMods; ut != nil {
		f.Mods = ut.XferMods(leader.Mods)
	}
}

// minionGroup is MONSTER_SpawnMinionGroup (0x59e7a0): min..max minions of the
// leader's minion class on the ring of 3 (flags 0x40, so no party pack of their
// own), skipped for champions. The count is Roll(max-min+1)+min on the
// leader's seed.
func (g *Game) minionGroup(w World, room *Room, c *Cell, leader *Unit, lo, hi int, pop *Population) {
	if leader.Champion {
		return
	}

	class := g.minionClass(leader.Class)
	if class < 0 || class >= len(g.Tables.Mons) {
		return
	}

	n := int(leader.Seed.Roll(int32(hi-lo+1))) + lo

	for ; n > 0; n-- {
		if m := g.follower(w, room, c, leader, class, 3, pop); m != nil {
			g.link(leader, m)
		}
	}
}

// partyPack is MONSTER_SpawnMinionPackForLeader (0x5b0420), run by the unit
// creation of every monster made with flags 0: for a class with a valid minion1
// it draws PartyMin..PartyMax (RAND_RollRangeFromSeed on the unit's seed) units
// alternating minion1 / minion2 on the ring of 4 (flags 0x40). Only a SetBoss
// class links them to the unit. Not modelled: the branch for a minion1 whose
// base class is 0x105 (it calls MONSTER_SpawnMonsterWithMinionsNearUnit).
func (g *Game) partyPack(w World, room *Room, c *Cell, u *Unit, class int, pop *Population) {
	if !g.FullPacks || class < 0 || class >= len(g.Tables.Mons) {
		return
	}

	m := &g.Tables.Mons[class]
	if m.Minion1 < 0 || m.Minion1 >= len(g.Tables.Mons) {
		return
	}

	n := m.PartyMin
	if m.PartyMax > m.PartyMin {
		n = int(u.Seed.Roll(int32(m.PartyMax-m.PartyMin+1))) + m.PartyMin
	}

	last := 0
	if m.Minion2 >= 0 && m.Minion2 < len(g.Tables.Mons) {
		last = 1
	}

	cls := [2]int{m.Minion1, m.Minion2}
	idx := 0

	for ; n > 0; n-- {
		if f := g.follower(w, room, c, u, cls[idx], 4, pop); f != nil {
			f.Party = true

			if !m.Has(FlagSetBoss) {
				f.Minion, f.Leader = false, nil
			}
		}

		idx++
		if idx > last {
			idx = 0
		}
	}
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
		// FUN_0059fc40 after FUN_0059e330: a rare gets 3..6 minion1 units
		if g.FullPacks {
			g.minionGroup(w, room, c, u, 3, 6, pop)
		}

		return true
	}

	// FUN_0054c050: 1..3 champion minions on the ring of radius 4
	n := int(u.Seed.Roll(3)) + 1

	for ; n > 0; n-- {
		if m := g.follower(w, room, c, u, class, 4, pop); m != nil {
			m.Champion = true
			g.link(u, m)
			g.partyPack(w, room, c, m, class, pop) // spawned with flags 0
		}
	}

	return true
}
