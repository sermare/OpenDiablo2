package d2monreg

// MONSTER_SpawnSuperUnique (0x5a2480): the creation of a superuniques.txt boss
// from a preset node (the class draw and spot are FUN_0054c470, see
// preset.go). Read from the decompilation and the disassembly.
//
// VERIFIED: refused above Hell and for a hcIdx already made in this game
// (unless the Stacks column); flag 2 (super unique) and the super unique id; the
// modifier list = Mod1..Mod3 of the row (reading stops at the first 0, id 24 is
// skipped, id 30 is remembered for the spectral-hit init) + one extra
// non-champion pick per difficulty step (Nightmare 1, Hell 2) + id 22 appended
// last; then MONSTER_RunUModInitCallbacks with a minion group of
// MinGrp..MaxGrp units (both +difficulty when both are non-zero) of the class'
// minion1 (the class itself when it has none: the Countess' six are
// corruptrogue3), ring 3, flags 0x40; then the extras of a few hcIdx
// (below). The picks and the group count use the unit's own seed.
//
// UNVERIFIED: AutoPos rows are placed on the preset's spot here (the exe zeroes
// the position and lets MONAI_SpawnAtRandomPointInRoom find one); hcIdx 60's
// extras (class computed by MONSTER_ComputeSpawnRangeForLevel) are not made;
// the quest hooks of hcIdx 6, 26, 27, 29, 36..39, 43..45 are not modelled.

// SuperRec is the part of a superuniques.txt row the creation reads.
type SuperRec struct {
	HcIdx          int
	Class          int
	Mods           [3]int
	MinGrp, MaxGrp int
	Stacks         bool
}

// ModSpectralHit is the monumod id whose init callback the exe runs after
// the super unique's group (MONSTER_ApplyUModSpectralHit).
const ModSpectralHit = 30

// modQuestComplete is the id appended last to every super unique.
const modQuestComplete = 22

// superExtra is a group the exe adds around a super unique of a given hcIdx
// (the switch at the end of 0x5a2480).
type superExtra struct {
	class  int
	n      int // fixed count
	rolled int // when > 0 the count is Roll(rolled) + n
	ring   int
	link   bool // MONAI_AddMinionToLeader (MONSTER_SpawnMonsterGroupNearUnit)
}

var superExtras = map[int][]superExtra{
	10: {{class: 4, rolled: 5, n: 2, ring: 4}, {class: 0x114, n: 1, ring: 4}, {class: 0x17e, n: 1, ring: 4},
		{class: 0x181, n: 1, ring: 4}, {class: 0x185, n: 1, ring: 4}},
	0x2a: {{class: 0x1c5, n: 0x14, ring: 0x14, link: true}}, // Shenk: twenty Enslaved
	0x3e: {{class: 0x17d, n: 10, ring: 0x14, link: true}},
}

// SuperUnique creates one super unique at the preset spot (x, y). It returns nil
// when it is refused (difficulty, already made) or when no spot was found.
func (g *Game) SuperUnique(w World, room *Room, rec SuperRec, x, y int, pop *Population) *Unit {
	if g.Difficulty > 2 {
		return nil
	}

	if !rec.Stacks && g.superMade[rec.HcIdx] {
		return nil
	}

	u := g.PlacePreset(w, room, rec.Class, x, y, pop)
	if u == nil {
		return nil
	}

	if g.superMade == nil {
		g.superMade = map[int]bool{}
	}

	g.superMade[rec.HcIdx] = true
	u.Super = rec.HcIdx + 1

	used := map[int]bool{}

	for _, id := range rec.Mods {
		if id == 0 {
			break
		}

		if id != 0x18 {
			u.Mods = append(u.Mods, id)
			used[id] = true
		}
	}

	if ut := g.Tables.UMods; ut != nil && g.Difficulty > 0 {
		for i := 0; i < g.Difficulty; i++ {
			id := ut.PickNonBoss(g.Tables, &u.Seed, g.Difficulty, u.Class, g.Expansion, used)
			if id == 0 {
				break
			}

			u.Mods = append(u.Mods, id)
			used[id] = true
		}
	}

	lo, hi := rec.MinGrp, rec.MaxGrp
	if lo != 0 && hi != 0 {
		lo += g.Difficulty
		hi += g.Difficulty
	}

	// FUN_0059e7a0 returns when the count is not positive
	if hi-lo+1 > 0 || lo > 0 {
		n := lo
		if hi-lo+1 > 0 {
			n += int(u.Seed.Roll(int32(hi - lo + 1)))
		}

		cls := g.minionClass(u.Class)

		for ; n > 0 && cls >= 0 && cls < len(g.Tables.Mons); n-- {
			if m := g.follower(w, room, nil, u, cls, 3, pop); m != nil {
				g.link(u, m)
			}
		}
	}

	for _, e := range superExtras[rec.HcIdx] {
		n := e.n
		if e.rolled > 0 {
			n += int(u.Seed.Roll(int32(e.rolled)))
		}

		for ; n > 0; n-- {
			if m := g.follower(w, room, nil, u, e.class, e.ring, pop); m != nil {
				m.Minion = e.link

				if !e.link {
					m.Leader = nil
				}
			}
		}
	}

	if len(u.Mods) < MaxUMods {
		u.Mods = append(u.Mods, modQuestComplete)
	}

	return u
}
