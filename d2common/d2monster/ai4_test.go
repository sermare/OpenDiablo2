package d2monster

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// worldX wraps the fake world with the optional extensions the new AIs use.
type worldX struct {
	*fakeWorld
	unit    *Target // answers SourceFinder
	ally    *Target
	allyD   int
	corpse  *Target
	nCorpse int
}

func (w *worldX) UnitTarget(*Brain, uint32) (Target, int, bool) {
	if w.unit == nil {
		return Target{}, 0, false
	}

	return *w.unit, 1, true
}

func (w *worldX) NearestAlly(*Brain) (Target, int, bool) {
	if w.ally == nil {
		return Target{}, 0, false
	}

	return *w.ally, w.allyD, true
}

func (w *worldX) NearestCorpse(*Brain, []int, int) (Target, bool) {
	w.nCorpse++

	if w.corpse == nil {
		return Target{}, false
	}

	return *w.corpse, true
}

func withSkills(p *Profile, slots ...int) *Profile {
	for _, s := range slots {
		p.Skills[s] = SkillSlot{Name: "sk", Mode: ModeSkill1}
	}

	return p
}

func TestMonsterAI4(t *testing.T) {
	tgt := &Target{ID: 9, X: 120, Y: 100, Size: 1}

	cases := []struct {
		name    string
		prof    *Profile
		dist    int
		inRange bool
		setup   func(b *Brain, w *worldX)
		want    []string // action log; nil = only check wake
		wake    int      // expected Wake when no action
	}{
		{name: "DoomKnight walks", prof: profile("DoomKnight", 0, 0, 100, 0), dist: 20, want: []string{"walk-target/7"}},
		{name: "DoomKnight attacks", prof: profile("DoomKnight", 100, 0, 0, 0), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "DoomKnight stalls in range", prof: profile("DoomKnight", 0, 9, 0, 0), dist: 3, inRange: true, wake: 9},
		{name: "DoomKnight stalls out of range", prof: profile("DoomKnight", 0, 0, 0, 14), dist: 20, wake: 14},
		{name: "PantherWoman attacks", prof: profile("PantherWoman", 0, 100, 0, 5), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "PantherWoman approaches", prof: profile("PantherWoman", 100, 0, 0, 5), dist: 20, want: []string{"walk-target/7"}},
		{name: "PantherWoman regroups", prof: profile("PantherWoman", 0, 0, 10, 5), dist: 20,
			setup: func(_ *Brain, w *worldX) { w.ally = &Target{ID: 3, X: 140, Y: 100, Size: 1}; w.allyD = 30 },
			want:  []string{"walk-target/7"}},
		{name: "QuillRat melee", prof: profile("QuillRat", 10, 0, 0, 5), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "QuillRat spike shot", prof: profile("QuillRat", 30, 100, 0, 5), dist: 10, want: []string{"attack5"}},
		{name: "QuillRat aggressive shoots", prof: profile("QuillRat", 30, 0, 0, 5), dist: 10,
			setup: func(b *Brain, _ *worldX) { b.Aggressive = true }, want: []string{"attack5"}},
		{name: "QuillRat kites", prof: profile("QuillRat", 30, 0, 0, 5), dist: 10, want: []string{"walk-to(95,100)"}},
		{name: "QuillRat command", prof: profile("QuillRat", 30, 0, 0, 5), dist: 10,
			setup: func(b *Brain, w *worldX) { b.PushCommand(Command{Type: CmdAlert, Target: 4}); w.unit = tgt },
			want:  []string{"attack5"}},
		{name: "SandLeaper leaps", prof: withSkills(profile("SandLeaper", 100, 0, 0, 0), 0), dist: 3, want: []string{"cast0"}},
		{name: "SandLeaper bites", prof: profile("SandLeaper", 0, 100, 0, 0), dist: 6, inRange: true, want: []string{"attack5"}},
		{name: "SandRaider approaches", prof: profile("SandRaider", 0, 0, 0, 100, 100), dist: 20, want: []string{"walk-target/0"}},
		{name: "GreaterMummy hits", prof: profile("GreaterMummy", 100, 0, 0, 0, 10), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "GreaterMummy breath", prof: profile("GreaterMummy", 100, 0, 0, 0, 10), dist: 3, want: []string{"attack5"}},
		{name: "GreaterMummy raises", prof: withSkills(profile("GreaterMummy", 0, 0, 100, 0, 10), 1), dist: 9,
			setup: func(_ *Brain, w *worldX) { w.corpse = &Target{ID: 5, X: 110, Y: 100, Size: 1} },
			want:  []string{"cast1"}},
		{name: "GreaterMummy walks without corpse", prof: profile("GreaterMummy", 0, 0, 0, 0, 10), dist: 9, want: []string{"walk-target/3"}},
		{name: "Fetish walks", prof: profile("Fetish", 100, 5, 3, 50), dist: 9, want: []string{"walk-target/7"}},
		{name: "Fetish attacks", prof: profile("Fetish", 100, 5, 3, 50), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "Fetish follows commanded unit", prof: profile("Fetish", 100, 5, 3, 50), dist: 9,
			setup: func(b *Brain, w *worldX) { b.PushCommand(Command{Type: CmdFollow, Target: 2}); w.unit = tgt },
			want:  []string{"walk-target/0"}},
		{name: "FetishBlowgun command attack", prof: profile("FetishBlowgun", 30, 0), dist: 9,
			setup: func(b *Brain, w *worldX) { b.PushCommand(Command{Type: CmdAlert, Target: 2}); w.unit = tgt },
			want:  []string{"attack4"}},
		{name: "FetishBlowgun shoots", prof: profile("FetishBlowgun", 30, 0), dist: 8, inRange: true, want: []string{"attack4"}},
		{name: "FetishShaman heals buddy", prof: withSkills(profile("FetishShaman", 100, 0, 15, 0, 30), 2), dist: 9,
			setup: func(_ *Brain, w *worldX) { w.ally = &Target{ID: 3, X: 105, Y: 100, Size: 1}; w.allyD = 5 },
			want:  []string{"cast2"}},
		{name: "BatDemon takes off", prof: profile("BatDemon", 20, 20, 50, 50, 25), dist: 9, want: []string{"attack10"}},
		{name: "Megademon attacks", prof: profile("Megademon", 0, 0, 100, 0, 0, 40), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "Megademon approaches", prof: profile("Megademon", 0, 0, 0, 100, 0, 40), dist: 9, want: []string{"walk-target/7"}},
		{name: "AbyssKnight attacks", prof: profile("AbyssKnight", 20, 0, 100, 10, 8, 50, 50, 10), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "AbyssKnight approaches", prof: profile("AbyssKnight", 20, 0, 100, 10, 8, 50, 100, 10), dist: 12, want: []string{"walk-target/7"}},
		{name: "Succubus attacks", prof: profile("Succubus", 100, 0, 0, 10, 5, 5, 0, 0), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "Succubus walks", prof: profile("Succubus", 0, 100, 0, 10, 5, 5, 0, 0), dist: 12, want: []string{"walk-target/0"}},
		{name: "Succubus casts", prof: withSkills(profile("Succubus", 0, 0, 100, 20, 5, 5, 0, 0), 0), dist: 9, want: []string{"cast0"}},
		{name: "Minion attacks", prof: profile("Minion", 100, 0, 100, 5), dist: 3, inRange: true, want: []string{"attack4"}},
		{name: "SandMaggot bites", prof: profile("SandMaggot", 0, 0, 3, 100, 30), dist: 3, inRange: true, want: []string{"attack4"}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := &worldX{fakeWorld: newFake(c.dist, c.inRange)}
			w.target.X, w.target.Y = 100+c.dist, 100
			b := brainAt(c.prof)

			if c.setup != nil {
				c.setup(b, w)
			}

			if !Tick(w, b) {
				t.Fatal("think did not run")
			}

			if c.want != nil {
				if strings.Join(w.log, ",") != strings.Join(c.want, ",") {
					t.Fatalf("log = %v, want %v", w.log, c.want)
				}

				return
			}

			if len(w.log) != 0 || b.Wake != c.wake {
				t.Fatalf("want sleep %d, got log=%v wake=%d", c.wake, w.log, b.Wake)
			}
		})
	}
}

// TestFetishPhases walks the Fetish through attack, loop and flee.
func TestFetishPhases(t *testing.T) {
	w := newFake(3, true)
	b := brainAt(profile("Fetish", 100, 5, 1, 50))
	b.HPPercent = 100

	runOne(t, w, b)

	if b.Scratch[fetPhase] != 1 {
		t.Fatalf("phase after first attack = %d", b.Scratch[fetPhase])
	}

	for i := 0; i < 3 && b.Scratch[fetPhase] == 1; i++ {
		b.WakeNow(0)
		Tick(w, b)
	}

	if b.Scratch[fetPhase] != 2 {
		t.Fatalf("fetish facing a healthy foe should flee (phase %d, log %v)", b.Scratch[fetPhase], w.log)
	}
}

// TestNoCommonMonsterIdles drives every stand-in and ported archetype for many
// seeds and requires that each acts (not just sleeps) at least once.
func TestNoCommonMonsterIdles(t *testing.T) {
	names := append([]string{"QuillRat", "PantherWoman", "SandLeaper", "SandRaider", "SandMaggot", "GreaterMummy",
		"Fetish", "FetishBlowgun", "FetishShaman", "BatDemon", "DoomKnight", "AbyssKnight", "Megademon",
		"Succubus", "Minion"}, GenericAIs()...)

	for _, n := range names {
		d, ok := Lookup(n)
		if !ok || !d.Implemented {
			t.Errorf("%s not registered", n)

			continue
		}

		if k, generic := genericAIs[n]; generic && (k == kindInert || k == kindPet) {
			continue
		}

		acted := false

		for id := uint32(1); id < 60 && !acted; id++ {
			w := &worldX{fakeWorld: newFake(6, id%2 == 0)}
			p := withSkills(profile(n, 60, 60, 60, 60, 20, 20, 20, 20), 0, 1, 2, 3)
			class := 1
			if n == "Ancient" { // dispatches on its class id (0x21c..0x21e)
				class = ancientC
			}

			b := NewBrain(id, class, Normal, p, testSeed)
			b.X, b.Y = 100, 100

			Tick(w, b)

			acted = len(w.log) > 0
		}

		if !acted {
			t.Errorf("%s never acted in 60 seeds", n)
		}
	}
}

// TestMonaiTableCoverage compares the real monai.txt names with the registry.
func TestMonaiTableCoverage(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	buf, err := os.ReadFile(filepath.Join(dir, "monsters", "patch_d2", "monai.txt"))
	if err != nil {
		t.Skip(err)
	}

	skip := map[string]bool{}
	for _, n := range DeliberatelyUnported {
		skip[strings.ToLower(n)] = true
	}

	for i, line := range strings.Split(string(buf), "\n") {
		name := strings.TrimSpace(strings.SplitN(line, "\t", 2)[0])
		if i == 0 || name == "" || skip[strings.ToLower(name)] {
			continue
		}

		if d, ok := Lookup(name); !ok || !d.Implemented {
			t.Errorf("monai AI %q has no Go implementation", name)
		}
	}
}
