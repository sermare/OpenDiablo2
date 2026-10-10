package d2drop

import (
	"fmt"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// TestCreateWithProperties creates whole items, properties included, and
// compares everything with ITEMGEN_CreateItemFromRequest of the real game:
// quality, flags, affix ids, unique / set row, every stat write with the list
// it went to, both generators and the one-per-game bitmask. Lord of
// Destruction games (version 100) and classic ones (version 0).
func TestCreateWithProperties(t *testing.T) {
	f := readCFile(t)
	c := cLoadCreator(t)
	bad, ok, refused := 0, 0, 0
	byQ := map[int]int{}

	for i := range f.Cases {
		cs := &f.Cases[i]
		if cs.T != "full" {
			continue
		}

		game := &GameState{Ladder: cs.Lad != 0}
		for _, r := range cs.M0 {
			game.markSpawned(r)
		}

		req := Request{
			Code: cs.C, ILvl: cs.IL, Quality: Quality(cs.Q), Difficulty: cs.Df, Version: cs.V, Expansion: cs.X == 1,
			ForcedID: cs.Fid, Flags: RequestFlags(cs.Fl), GameSeed: d2rand.Seed{Lo: cs.GS, Hi: 0x29a}, Game: game,
		}

		got, err := c.Create(req)
		if cs.O.OK == 0 {
			refused++

			if err == nil {
				bad++
				t.Errorf("case %d %+v: created, the game refuses", i, req)
			}

			continue
		}

		if err != nil {
			bad++
			if bad <= 10 {
				t.Errorf("case %d %+v: %v, the game creates it", i, req, err)
			}

			continue
		}

		ok++
		byQ[int(got.Quality)]++

		var want []string

		chk := func(name string, a, b interface{}) {
			if fmt.Sprint(a) != fmt.Sprint(b) {
				want = append(want, fmt.Sprintf("%s got %v want %v", name, a, b))
			}
		}

		var wantW []StatWrite

		for k, w := range cs.O.W {
			wantW = append(wantW, StatWrite{Kind: w[0].(string)[0], Stat: cNum(w[1]), Value: cNum(w[2]), Param: cNum(w[3]), List: cs.O.Ws[k]})
		}

		var m1 []int

		for r := 0; r < 0x1000; r++ {
			if game.spawned(r) && !cContains(cs.M0, r) {
				m1 = append(m1, r)
			}
		}

		chk("quality", int(got.Quality), cs.O.Q)
		chk("flags", fmt.Sprintf("%#x", got.Flags), fmt.Sprintf("%#x", cs.O.FL))
		chk("ilvl", got.ILvl, cs.O.IL)
		chk("prefix", got.Prefix, cs.O.Pre)
		chk("suffix", got.Suffix, cs.O.Suf)
		chk("auto", got.Auto, cs.O.Au)
		chk("uid", got.Unique, cs.O.Uid)
		chk("gfx", got.Gfx, cs.O.Gfx)
		chk("writes", cFmtWrites(got.Writes), cFmtWrites(wantW))
		chk("unit seed", [2]uint32{got.UnitSeed.Lo, got.UnitSeed.Hi}, cs.O.Us)
		chk("item seed", [2]uint32{got.ItemSeed.Lo, got.ItemSeed.Hi}, cs.O.Is)
		chk("mask", m1, cs.O.M1)

		if len(want) > 0 {
			bad++

			if bad <= 10 {
				t.Errorf("case %d %s ilvl %d q %d diff %d fid %d flags %#x mask %v seed %d:\n\t%s", i, cs.C, cs.IL, cs.Q, cs.Df, cs.Fid, cs.Fl, cs.M0, cs.GS,
					strings.Join(want, "\n\t"))
			}
		}
	}

	if ok+refused == 0 {
		t.Skip("no full cases")
	}

	t.Logf("%d items compared (by quality %v), %d refused by the game, %d mismatches", ok, byQ, refused, bad)

	if bad > 0 {
		t.Fail()
	}
}
