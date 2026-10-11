package d2monster

import "testing"

type wispWorld struct {
	*fb1World
	mates  []WispMate
	waits  []int
	blessN []int
}

func (w *wispWorld) WispFormation(*Brain) []WispMate { return w.mates }

func (w *wispWorld) WispBless(_ *Brain, _ Target, stat, value, frames int) {
	w.blessN = append(w.blessN, stat, value, frames)
}

func newWisp(frame int) (*wispWorld, *Brain) {
	p := withSkills(profile("WillOWisp", 0, 0, 0), 0, 1, 2, 3, 4)
	b := brainAt(p)
	w := &wispWorld{fb1World: newFB1(10, false)}
	w.frame = frame

	return w, b
}

// the fixture target stands at (110, 100)
func wispAt(b *Brain, off Point) { b.X, b.Y = 110+off.X, 100+off.Y }

func TestWispFormationSearch(t *testing.T) {
	for _, tc := range []struct {
		n, state int
	}{{4, 6}, {5, 5}} {
		w, b := newWisp(100)
		b.Scratch[0] = 4

		for i := 0; i < tc.n; i++ {
			mb := brainAt(b.Profile)
			w.mates = append(w.mates, WispMate{B: mb, Wait: func(f int) { w.waits = append(w.waits, f) }})
		}

		fb1Think(w, b, 10, false, true)

		for i, m := range w.mates {
			if m.B.Scratch != [3]int{tc.state, 0, i + 1} {
				t.Errorf("%d wisps: mate %d scratch %v", tc.n, i, m.B.Scratch)
			}
		}

		want := 0x151 - 100%0x151
		if len(w.waits) != tc.n || w.waits[0] != want || b.Wake != 100+want {
			t.Errorf("%d wisps: waits %v wake %d, want sync wait %d", tc.n, w.waits, b.Wake, want)
		}
	}

	w, b := newWisp(100)
	b.Scratch[0] = 4
	w.mates = []WispMate{{B: brainAt(b.Profile)}, {B: brainAt(b.Profile)}, {B: brainAt(b.Profile)}}
	fb1Think(w, b, 10, false, true)

	if b.Scratch[0] != 2 || b.Wake != 108 || b.Scratch[2] != 0 {
		t.Errorf("3 wisps: scratch %v wake %d, want state 2, wait 8, no cooldown", b.Scratch, b.Wake)
	}
}

func TestWispDanceFiveActsOnTheBeat(t *testing.T) {
	w, b := newWisp(0x43 * 3)
	b.Scratch = [3]int{5, 0, 1}
	wispAt(b, wispSlotsFive[1])
	fb1Think(w, b, 10, false, true)

	if len(w.log) != 1 || w.log[0] != "attack7" || b.Scratch[1] != 1 {
		t.Errorf("log %v scratch %v: want one cast (mode 7) and a counted step", w.log, b.Scratch)
	}

	// off the beat it waits for the next one
	w, b = newWisp(0x43*3 + 5)
	b.Scratch = [3]int{5, 0, 1}
	wispAt(b, wispSlotsFive[1])
	fb1Think(w, b, 10, false, true)

	if len(w.log) != 0 || b.Wake != w.frame+0x43-5 {
		t.Errorf("off beat: log %v wake %d", w.log, b.Wake)
	}
}

func TestWispDanceFiveEnds(t *testing.T) {
	w, b := newWisp(0x43 * 3)
	b.Scratch = [3]int{5, 3, 2}
	wispAt(b, wispSlotsFive[2])
	fb1Think(w, b, 10, false, true)

	if b.Scratch[0] != 2 || b.Scratch[2] != w.frame+0x708 || b.Wake != w.frame+0x151-w.frame%0x151 {
		t.Errorf("scratch %v wake %d", b.Scratch, b.Wake)
	}

	if len(w.blessN) != 0 {
		t.Error("the five-wisp dance gives no stat")
	}
}

func TestWispDanceSixAdvancesAndBlesses(t *testing.T) {
	w, b := newWisp(0x43 * 3)
	b.Scratch = [3]int{6, 3, 1}
	wispAt(b, wispSlotsSix[0])
	fb1Think(w, b, 10, false, true)

	if b.Scratch != [3]int{7, 0, 1} {
		t.Errorf("scratch %v, want phase 7", b.Scratch)
	}

	if len(w.blessN) != 3 || w.blessN[0] != 0x50 || w.blessN[1] != 50 || w.blessN[2] != 0x1a5e00 {
		t.Errorf("bless %v, want stat 0x50 value 50 for 0x1a5e00 frames", w.blessN)
	}

	// the last phase ends the dance and still blesses
	w, b = newWisp(0x43 * 3)
	b.Scratch = [3]int{12, 3, 4}
	wispAt(b, wispSlotsSix[27])
	fb1Think(w, b, 10, false, true)

	if b.Scratch[0] != 2 || b.Scratch[2] != w.frame+0x708 || len(w.blessN) != 3 {
		t.Errorf("end: scratch %v bless %v", b.Scratch, w.blessN)
	}
}

func TestWispDanceSixZeroActOnlyCounts(t *testing.T) {
	w, b := newWisp(0x43 * 3)
	b.Scratch = [3]int{12, 0, 4} // act table entry 27 is (0,0)
	wispAt(b, wispSlotsSix[27])
	fb1Think(w, b, 10, false, true)

	if len(w.log) != 0 || b.Scratch[1] != 1 || b.Wake != w.frame+0x43 {
		t.Errorf("log %v scratch %v wake %d", w.log, b.Scratch, b.Wake)
	}
}

func TestWispDanceTravelsToTheSlot(t *testing.T) {
	for id := uint32(1); id < 200; id++ {
		w, b := newWisp(1)
		b.ID = id
		b.Seed.Init(id)
		b.Scratch = [3]int{6, 0, 2}
		b.X, b.Y = 100, 100 // far from the slot

		s := shadow(b)
		if s.Roll(100) < 0x22 {
			continue // the wander branch
		}

		fb1Think(w, b, 10, false, true)

		// state 6 slot 2 is table index 1: (1,-5) from the target (110,100)
		if len(w.log) != 1 || w.log[0] != "walk-to(111,95)" {
			t.Errorf("log %v, want a walk to the slot point", w.log)
		}

		return
	}

	t.Skip("no seed avoided the wander branch")
}

func TestWispDanceOutsideFormationStops(t *testing.T) {
	w, b := newWisp(10)
	b.Scratch = [3]int{6, 0, 5}
	fb1Think(w, b, 10, false, true)

	if b.Scratch[0] != 2 || b.Wake != 18 {
		t.Errorf("scratch %v wake %d", b.Scratch, b.Wake)
	}
}

func TestWispTables(t *testing.T) {
	if len(wispSlotsSix) != 28 || len(wispActSix) != 28 {
		t.Fatal("seven phases of four wisps")
	}

	if wispActSix[27] != (Point{}) {
		t.Error("the last act entry is the exe's (0,0)")
	}

	// the second five-wisp table is the first rotated by two slots
	for s := 1; s <= 5; s++ {
		if wispActFive[s] != wispSlotsFive[(s+1)%5+1] {
			t.Errorf("slot %d: act %v", s, wispActFive[s])
		}
	}
}
