package d2monreg

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// MaxLevelTypes is the number of monster types a level region may hold.
const MaxLevelTypes = 13

// TypeEntry is one monster type of a level (the 0x34-byte entry of the
// MonsterRegion: class, rarity, up to three look variants).
type TypeEntry struct {
	Class    int
	Rarity   int
	Variants [][16]byte
}

// Region is the per-level monster record MONREGION_BuildLevelMonsterRegions
// makes for a game (0x2e4 bytes in memory; offsets in the comments).
type Region struct {
	Act, Level int
	Types      []TypeEntry // +0x10 count, +0x14 entries
	// TotalRarity is the byte at +0x11 (sum of the rarities, a byte) and N2 the
	// byte at +0x12.
	TotalRarity, N2 int
	// RoomsSeen (+4) counts the rooms that asked for a population, Placed (+8)
	// the rooms where something was created.
	RoomsSeen, Placed int
	// TotalRooms (+0xc) is the number of populated rooms of the level, -1 until
	// the first room asks.
	TotalRooms int
	Density    int // +0x2b8 (MonDen of the difficulty)
	UMin, UMax int // +0x2bc, +0x2bd
	Wndr       int // +0x2be
	Uniques    int // +0x2c8: uniques made so far
	Wanderers  int // +0x2c4: wandering monsters made so far
	Spawned    int // +0x2cc: monsters counted by FUN_00545ce0
	Quest      int // +0x2d8
	MonLvl     int // +0x2dc / +0x2e0
}

// BuildRegions is MONREGION_InitForGame: one game-seed step gives the region
// seed, then every level 1..LevelCount-1 gets its monster types (a random draw
// without replacement from the mon / nmon list) and their look variants. diff
// is the game difficulty and expansion the flag at game+0x70.
//
// Verified against the emulated game for 8 seeds x 3 difficulties x 2 modes
// (all types, rarities, variants and counters of all levels).
func BuildRegions(t *Tables, game *d2rand.Seed, diff int, expansion bool) (regs []*Region, regionSeed uint32) {
	regionSeed = game.Step()
	s := d2rand.New(regionSeed)
	regs = make([]*Region, t.LevelCount)

	for lvl := 1; lvl < t.LevelCount; lvl++ {
		rec := t.Level(lvl)
		if rec == nil {
			continue
		}

		r := &Region{Act: rec.Act, Level: lvl, Density: rec.MonDen[diff], UMin: rec.MonUMin[diff], UMax: rec.MonUMax[diff],
			Wndr: rec.MonWndr, Quest: rec.Quest, TotalRooms: -1}

		ex := 0
		if expansion {
			ex = 1
		}

		r.MonLvl = rec.MonLvl[ex][diff]

		pickTypes(t, rec, r, s, diff != 0)

		for i := range r.Types {
			rollTransforms(t, s, &r.Types[i], 3)
		}

		regs[lvl] = r
	}

	return regs, regionSeed
}

// pickTypes is MONREGION_PickLevelMonsterTypes (0x5454d0).
func pickTypes(t *Tables, rec *Level, r *Region, s *d2rand.Seed, nm bool) {
	limit := rec.NumMon
	if limit > MaxLevelTypes {
		limit = MaxLevelTypes
	}

	list := rec.Mon[:rec.CountMon]
	count := rec.CountMon

	if nm {
		list = rec.NMon[:rec.CountNMon]
		count = rec.CountNMon
	}

	if count < limit {
		limit = count
	}

	pool := append([]int(nil), list...)
	left := count

	ranged := func(class int) bool {
		return class >= 0 && class < len(t.Mons) && t.Mons[class].Has(FlagRangedType)
	}

	for i := 0; i < limit; i++ {
		if left < 1 {
			break
		}

		idx := int(s.Roll(int32(left)))
		cls := pool[idx]

		if i == 0 && rec.RangedSpawn != 0 {
			// up to four blocks of five re-draws while the class is not a ranged one
			for tries := 0; tries < 0x14; tries += 5 {
				if ranged(cls) {
					break
				}

				found := false

				for k := 0; k < 5; k++ {
					idx = int(s.Roll(int32(left)))
					cls = pool[idx]

					if k < 4 && ranged(cls) {
						found = true
						break
					}
				}

				if found {
					break
				}
			}
		}

		left--
		if idx < left {
			copy(pool[idx:], pool[idx+1:left+1])
		}

		if cls >= 0 && cls < len(t.Mons) && t.Mons[cls].Has(FlagIsSpawn) {
			r.Types = append(r.Types, TypeEntry{Class: cls, Rarity: t.Mons[cls].Rarity})
			r.TotalRarity = (r.TotalRarity + t.Mons[cls].Rarity) & 0xff
			r.N2++
		}
	}
}

// rollTransforms is MONREGION_RollRandomTransforms (0x5bb680): it gives a
// type up to n distinct look variants (a variant is a random index per
// component, bounded by the component counts of the monster's monstats2 row).
func rollTransforms(t *Tables, s *d2rand.Seed, e *TypeEntry, n int) {
	if e.Class < 0 || e.Class >= len(t.Mons) {
		return
	}

	ex := t.Mons[e.Class].Ex
	if ex < 0 || ex >= len(t.Mon2s) {
		return
	}

	m2 := &t.Mon2s[ex]
	have := len(e.Variants)

	lim := int(int32(1)<<uint(m2.Bits&31)) - have
	if n > lim {
		n = lim
	}

	if n <= 0 {
		return
	}

	if have+n > 3 {
		n = 3 - have
	}

	if n <= 0 {
		return
	}

	roll := func(c int) byte {
		if c <= 0 {
			return 0
		}

		return byte(s.Roll(int32(c)))
	}

	if have == 0 {
		var v [16]byte

		for i := 0; i < 16; i++ {
			if m2.Counts[i] <= 1 {
				v[i] = 0
			} else {
				v[i] = roll(m2.Counts[i])
			}
		}

		e.Variants = append(e.Variants, v)
		n--
	}

	if n == 0 {
		return
	}

	var comps []int

	for i := 0; i < 16; i++ {
		if m2.Counts[i] > 1 {
			comps = append(comps, i)
		}
	}

	var a, b int

	switch {
	case len(comps) == 1:
		a, b = comps[0], comps[0]
	case len(comps) <= 0:
		return
	default:
		k := len(comps)
		r1 := int(s.Roll(int32(k)))
		a = comps[r1]
		comps[r1] = comps[k-1]
		k--
		r2 := int(s.Roll(int32(k)))
		b = comps[r2]
	}

	for ; n > 0; n-- {
		v := e.Variants[0]
		tries := 3

		for {
			v[a] = roll(m2.Counts[a])

			if a != b {
				v[b] = roll(m2.Counts[b])
			}

			dup := false

			for _, o := range e.Variants {
				if o == v {
					tries--
					dup = true
				}
			}

			if dup && tries != 0 {
				continue
			}

			break
		}

		e.Variants = append(e.Variants, v)
	}
}
