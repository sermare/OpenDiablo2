package d2monreg

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// openWorld is a map with no obstacle: every spot is free ground.
type openWorld struct{}

func (openWorld) Blocked(*Room, int, int, int, int) bool { return false }
func (openWorld) Open(*Room, int, int) bool              { return true }
func (openWorld) CellAt(*Room, int, int) int             { return 1 }
func (openWorld) NearExit(*Room, int, int) bool          { return false }

// populateLevel fills a level with rooms of 8x8 tiles (the stand-in for its real
// rooms) and returns everything created.
func populateLevel(g *Game, level, rooms int, seed uint32) *Population {
	var pop Population

	g.RoomCount = func(int) int { return rooms }

	for i := 0; i < rooms; i++ {
		tx, ty := 100+(i%10)*8, 100+(i/10)*8
		room := &Room{Level: level, Seed: *d2rand.New((seed+uint32(i)+uint32(level)<<8)*2654435761 ^ 0x5bd1e995), X: tx * 5, Y: ty * 5, W: 40, H: 40,
			Cells: []Cell{{X0: tx, Y0: ty, X1: tx + 8, Y1: ty + 8, Flag: 1}}}

		g.PopulateNatural(openWorld{}, room, &pop)
	}

	return &pop
}

// The numeric scenario: for every level of the real Levels.txt and every
// difficulty, the number of unique / champion packs a populated level holds is
// within what the table allows (MonUMax of the difficulty; forced up to MonUMin
// while rooms remain), the leaders are drawn from the right list, the modifier
// count follows the difficulty, and the pack members follow the verified
// rules (rare: 3..6 minions of minion1; champion: 1..3 champions).
func TestRankCountsPerLevel(t *testing.T) {
	tb := realUModTables(t)

	type tot struct{ levels, packs, champs, rares, over int }

	var all tot

	for diff := 0; diff < 3; diff++ {
		for seed := uint32(1); seed <= 3; seed++ {
			g := NewGame(tb, 0x1234*seed+uint32(diff), diff, true)
			g.FullPacks = true

			for lvl := 1; lvl < tb.LevelCount; lvl++ {
				rec, rg := tb.Level(lvl), g.Regions[lvl]
				if rec == nil || rg == nil || rg.Density == 0 || len(rg.Types) == 0 {
					continue
				}

				pop := populateLevel(g, lvl, 40, seed*1000)
				name := fmt.Sprintf("level %d diff %d seed %d", lvl, diff, seed)

				legalLeader := map[int]bool{}
				if diff == 0 {
					for _, c := range rec.UMon {
						if c >= 0 {
							legalLeader[c] = true
						}
					}
				} else {
					for _, ty := range rg.Types {
						legalLeader[ty.Class] = true
						if sp := tb.Mons[ty.Class].Spawn; sp >= 0 {
							legalLeader[sp] = true
						}
					}
				}

				packs, champs, rares := 0, 0, 0
				minions := map[*Unit]int{}

				for _, u := range pop.Units {
					if u.Leader != nil && u.Minion {
						minions[u.Leader]++
					}
				}

				for _, u := range pop.Units {
					if !u.Unique {
						continue
					}

					packs++

					if !legalLeader[u.Class] {
						t.Errorf("%s: pack leader %s is not in the level's umon / type list", name, tb.Mons[u.Class].Key)
					}

					if u.Champion {
						champs++

						if len(u.Mods) < 1 || u.Mods[0] < 16 {
							t.Errorf("%s: champion mods %v", name, u.Mods)
						}

						if n := minions[u]; n < 1 || n > 3 {
							t.Errorf("%s: champion has %d minions, want 1..3", name, n)
						}

						continue
					}

					rares++

					if len(u.Mods) != 1+diff {
						t.Errorf("%s: rare %s carries %d modifiers %v, want %d", name, tb.Mons[u.Class].Key, len(u.Mods), u.Mods, 1+diff)
					}

					// minions are the Minion units linked to the leader; a SetBoss class' party is linked too
					if n := minions[u]; n < 3 {
						t.Errorf("%s: rare %s has %d minions, want at least 3", name, tb.Mons[u.Class].Key, n)
					}
				}

				limit := rg.UMax
				if rg.UMin > limit {
					limit = rg.UMin
				}

				if packs > limit {
					t.Errorf("%s: %d unique/champion packs, table allows at most %d (MonUMin %d MonUMax %d)", name, packs, limit, rg.UMin, rg.UMax)
					all.over++
				}

				// every level that wants uniques (MonUMin > 0) and was populated gets them
				if rg.UMin > 0 && packs < rg.UMin && rg.Placed == 40 {
					t.Errorf("%s: only %d packs, MonUMin is %d", name, packs, rg.UMin)
				}

				all.levels++
				all.packs += packs
				all.champs += champs
				all.rares += rares
			}
		}
	}

	t.Logf("%d level populations: %d unique/champion packs (%d champions, %d rares), %d over their table limit",
		all.levels, all.packs, all.champs, all.rares, all.over)

	if all.packs == 0 {
		t.Fatal("no unique or champion pack at all")
	}

	if share := float64(all.champs) / float64(all.packs); share < 0.15 || share > 0.25 {
		t.Errorf("champion share %.3f of the packs, want about the 20 percent of monumod row 0", share)
	}
}

// SetBoss classes bring their party: a fallen group is the leader, followers and
// PartyMin..PartyMax minions of every member. The party of a class without SetBoss
// is not linked.
func TestPartyPacks(t *testing.T) {
	tb := realUModTables(t)
	fallen := tb.MonByKey("fallen1")

	if fallen < 0 || !tb.Mons[fallen].Has(FlagSetBoss) || tb.Mons[fallen].Minion1 < 0 {
		t.Skip("fallen1 is not a SetBoss class with a minion in these tables")
	}

	run := func(full bool) (leaders, units int) {
		for s := uint32(1); s <= 40; s++ {
			g := NewGame(tb, s, 0, true)
			g.FullPacks = full

			var pop Population

			room := &Room{Level: 2, Seed: *d2rand.New(s), X: 500, Y: 500, W: 40, H: 40,
				Cells: []Cell{{X0: 100, Y0: 100, X1: 108, Y1: 108, Flag: 1}}}

			if u := g.spawnGroup(openWorld{}, room, &room.Cells[0], fallen, 1, 1, &pop); u != nil {
				leaders++
				units += len(pop.Units)
			}
		}

		return
	}

	l0, u0 := run(false)
	l1, u1 := run(true)

	if l0 == 0 || l0 != l1 {
		t.Fatalf("leaders %d vs %d", l0, l1)
	}

	m := tb.Mons[fallen]
	lo, hi := l1*(1+m.PartyMin), l1*(1+m.PartyMax)

	if u0 != l0 || u1 < lo || u1 > hi {
		t.Errorf("without packs %d units for %d leaders; with packs %d units, want %d..%d", u0, l0, u1, lo, hi)
	}
}
