package d2drop

import (
	"fmt"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// TestPropertyFunctions replays single property functions (table 742620)
// with arbitrary instances against the game: writes, return value, item
// generator and flags.
func TestPropertyFunctions(t *testing.T) {
	f := readCFile(t)
	c := cLoadCreator(t)
	bad, n := 0, 0

	for i := range f.Cases {
		cs := &f.Cases[i]
		if cs.T != "fn" {
			continue
		}

		n++

		st := newPropState(c, cs)
		pre := len(st.writes)
		pc := &propCtx{c: c, st: st, kind: cs.K, sel: cs.Sel}
		inst := &PropInst{Prop: 0, Param: cs.I[0], Min: cs.I[1], Max: cs.I[2]}

		ret := propFuncs[cs.F](pc, propArgs{inst: inst, set: cs.Set, stat: cs.St, val: cs.Val, prev: cs.Pv})

		got, want := st.writes[pre:], cs.want()
		wantSeed := d2rand.Seed{Lo: cs.SA[0], Hi: cs.SA[1]}

		wantRet := -999
		if cs.Ret != nil {
			wantRet = *cs.Ret
		}

		if cFmtWrites(got) != cFmtWrites(want) || st.item != wantSeed || ret != wantRet || st.flags != cs.Fl2 {
			bad++

			if bad <= 12 {
				t.Errorf("case %d f%d kind %d %s ilvl %d inst %v set %d stat %#x val %d prev %d:\n got %s seed %v ret %d flags %#x\nwant %s seed %v ret %d flags %#x",
					i, cs.F, cs.K, cs.C, cs.IL, cs.I, cs.Set, cs.St, cs.Val, cs.Pv, cFmtWrites(got), st.item, ret, st.flags, cFmtWrites(want), wantSeed, wantRet, cs.Fl2)
			}
		}
	}

	if n == 0 {
		t.Skip("no function cases")
	}

	t.Logf("%d property function calls, %d mismatches", n, bad)

	if bad > 0 {
		t.Fatalf("%d of %d property function calls differ from the game", bad, n)
	}
}

// TestPickUniqueSet replays PickUniqueItem and PickSetItem against the game:
// the row chosen, what the pick writes, the item generator and the
// one-per-game bitmask.
func TestPickUniqueSet(t *testing.T) {
	f := readCFile(t)
	c := cLoadCreator(t)
	bad, n := 0, 0

	for i := range f.Cases {
		cs := &f.Cases[i]
		if cs.T != "pick" {
			continue
		}

		n++

		cc := c

		if len(cs.Ov) > 0 {
			ov := *cc
			ut := *cc.Uniques
			ut.Uniques = append([]UniqueItem(nil), cc.Uniques.Uniques...)
			ov.Uniques = &ut

			for _, o := range cs.Ov {
				u := &ut.Uniques[o[0]]
				if o[1]&1 != 0 {
					u.Enabled = !u.Enabled
				}

				if o[1]&2 != 0 {
					u.NoLimit = !u.NoLimit
				}

				if o[1]&8 != 0 {
					u.Ladder = !u.Ladder
				}
			}

			cc = &ov
		}

		st := newPropState(cc, cs)
		pre := len(st.writes)
		st.req.ForcedID = cs.Fid
		st.req.Flags = RequestFlags(cs.Rf)
		st.req.Version = cs.Iv
		game := &GameState{Ladder: cs.Lad != 0}

		for _, r := range cs.M0 {
			game.markSpawned(r)
		}

		st.req.Game = game
		st.uniqueRow = -1

		var ok bool

		if cs.Q == int(QualityUnique) {
			ok = cc.pickUnique(st)
		} else {
			ok = cc.pickSet(st)
		}

		var m1 []int

		for r := 0; r < 0x1000; r++ {
			if game.spawned(r) && !cContains(cs.M0, r) {
				m1 = append(m1, r)
			}
		}

		got, want := st.writes[pre:], cs.want()
		wantSeed := d2rand.Seed{Lo: cs.SA[0], Hi: cs.SA[1]}
		wantOK := cs.Ret != nil && *cs.Ret != 0

		if cs.Q == int(QualitySet) && cs.Iv == 0 && !wantOK {
			// A failed classic set pick restores the item generator with a
			// high word taken from a caller register (an address): only the
			// low word is deterministic.
			st.item.Hi, wantSeed.Hi = 0, 0
		}

		if ok != wantOK || st.uniqueRow != cs.Uid || cFmtWrites(got) != cFmtWrites(want) || st.item != wantSeed ||
			st.flags != cs.Fl2 || fmt.Sprint(m1) != fmt.Sprint(append([]int(nil), cs.M1...)) {
			bad++

			if bad <= 12 {
				t.Errorf("case %d q%d %s ilvl %d ver %d fid %d flags %d ladder %d mask %v:\n got ok %v row %d spawned %v flags %#x seed %v%s\nwant ok %v row %d spawned %v flags %#x seed %v%s",
					i, cs.Q, cs.C, cs.IL, cs.Iv, cs.Fid, cs.Rf, cs.Lad, cs.M0, ok, st.uniqueRow, m1, st.flags, st.item, cFmtWrites(got),
					wantOK, cs.Uid, cs.M1, cs.Fl2, wantSeed, cFmtWrites(want))
			}
		}
	}

	if n == 0 {
		t.Skip("no pick cases")
	}

	t.Logf("%d unique / set picks, %d mismatches", n, bad)

	if bad > 0 {
		t.Fatalf("%d of %d picks differ from the game", bad, n)
	}
}

func cContains(l []int, x int) bool {
	for _, v := range l {
		if v == x {
			return true
		}
	}

	return false
}

// TestClassicPropertyFunctions replays the property handlers of classic items
// (version 0, table 741e80) with arbitrary instances against the game.
func TestClassicPropertyFunctions(t *testing.T) {
	f := readCFile(t)
	c := cLoadCreator(t)
	bad, n := 0, 0

	for i := range f.Cases {
		cs := &f.Cases[i]
		if cs.T != "cfn" {
			continue
		}

		n++

		st := newPropState(c, cs)
		pre := len(st.writes)
		pc := &propCtx{c: c, st: st, kind: cs.K, sel: cs.Sel}

		pc.applyClassic(&PropInst{Prop: cs.P, Param: cs.I[0], Min: cs.I[1], Max: cs.I[2]})

		got, want := st.writes[pre:], cs.want()
		wantSeed := d2rand.Seed{Lo: cs.SA[0], Hi: cs.SA[1]}

		if cFmtWrites(got) != cFmtWrites(want) || st.item != wantSeed || st.flags != cs.Fl2 {
			bad++

			if bad <= 12 {
				t.Errorf("case %d prop %d (%s) kind %d %s ilvl %d inst %v:\n got %s seed %v flags %#x\nwant %s seed %v flags %#x",
					i, cs.P, c.Props.Props[cs.P].Code, cs.K, cs.C, cs.IL, cs.I, cFmtWrites(got), st.item, st.flags, cFmtWrites(want), wantSeed, cs.Fl2)
			}
		}
	}

	if n == 0 {
		t.Skip("no classic function cases")
	}

	t.Logf("%d classic property function calls, %d mismatches", n, bad)

	if bad > 0 {
		t.Fatalf("%d of %d classic property function calls differ from the game", bad, n)
	}
}
