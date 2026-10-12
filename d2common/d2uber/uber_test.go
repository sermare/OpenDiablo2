package d2uber

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
)

func newEvent() (*d2boss.Manager, *Event, *[]string) {
	var logs []string

	m := d2boss.New(func(s string) { logs = append(logs, s) })
	e := New(m)

	return m, e, &logs
}

func kinds(as []d2boss.Action, k d2boss.ActionKind) []d2boss.Action {
	var out []d2boss.Action

	for _, a := range as {
		if a.Kind == k {
			out = append(out, a)
		}
	}

	return out
}

func TestKeyPortalAreas(t *testing.T) {
	// the exe tries the areas from a random start (0, 1 or 2) and takes the first one not opened yet; the fourth
	// portal of a game fails
	tests := []struct {
		name string
		pick func(int) int
		want []int
	}{
		{"start 0 (default)", nil, []int{133, 134, 135}},
		{"start 1", func(int) int { return 1 }, []int{134, 135, 133}},
		{"start 2", func(int) int { return 2 }, []int{135, 133, 134}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, e, _ := newEvent()
			e.Pick = tt.pick

			for i, want := range tt.want {
				as := e.OpenKeyPortal(m)
				p := kinds(as, d2boss.ActPortal)

				if len(p) != 1 || p[0].Level != want {
					t.Fatalf("portal %d: %+v, want level %d", i, as, want)
				}
			}

			if p := kinds(e.OpenKeyPortal(m), d2boss.ActPortal); len(p) != 0 {
				t.Fatalf("a fourth area portal opened: %+v", p)
			}
		})
	}
}

// Both portals open only in Harrogath on Hell, and the Tristram portal once per game (VERIFIED shape, D2MOO).
func TestCubePortalsNeedHarrogathOnHell(t *testing.T) {
	for _, c := range []struct {
		level, diff int
		ok          bool
	}{{LevelHarrogath, 2, true}, {LevelHarrogath, 1, false}, {LevelHarrogath, 0, false}, {1, 2, false}, {LevelTristram, 2, false}} {
		m, e, _ := newEvent()
		e.Where = func() (int, int) { return c.level, c.diff }

		keys := len(kinds(e.OpenKeyPortal(m), d2boss.ActPortal)) == 1
		finale := len(kinds(e.OpenFinalePortal(m), d2boss.ActPortal)) == 1

		if keys != c.ok || finale != c.ok {
			t.Errorf("level %d difficulty %d: keys portal %v, finale portal %v, want %v", c.level, c.diff, keys, finale, c.ok)
		}
	}

	m, e, _ := newEvent()
	e.Where = func() (int, int) { return LevelHarrogath, 2 }

	if len(kinds(e.OpenFinalePortal(m), d2boss.ActPortal)) != 1 || len(kinds(e.OpenFinalePortal(m), d2boss.ActPortal)) != 0 {
		t.Error("the Tristram portal must open once per game")
	}
}

func TestAreaBossesAndOrgans(t *testing.T) {
	tests := []struct {
		level, class int
		key, organ   string
	}{
		{LevelMatronsDen, ClassLilith, "uberandariel", CodeHorn},
		{LevelForgottenSands, ClassUberDuriel, "uberduriel", CodeBaalEye},
		{LevelFurnaceOfPain, ClassUberIzual, "uberizual", CodeMephBrain},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			m, _, _ := newEvent()

			sp := kinds(m.Enter(tt.level), d2boss.ActSpawnMonster)
			if len(sp) != 1 || sp[0].Key != tt.key || sp[0].Class != tt.class || !sp[0].Boss {
				t.Fatalf("spawn %+v", sp)
			}

			if again := kinds(m.Enter(tt.level), d2boss.ActSpawnMonster); len(again) != 0 {
				t.Fatalf("second entry spawned %+v", again)
			}

			dr := kinds(m.Killed(d2boss.Kill{Class: tt.class, Super: -1, Name: "x"}), d2boss.ActDropItem)
			if len(dr) != 1 || dr[0].Key != tt.organ {
				t.Fatalf("drop %+v, want %s", dr, tt.organ)
			}
		})
	}
}

func TestFinaleTristram(t *testing.T) {
	m, e, logs := newEvent()

	p := kinds(e.OpenFinalePortal(m), d2boss.ActPortal)
	if len(p) != 1 || p[0].Level != LevelTristram {
		t.Fatalf("finale portal %+v", p)
	}

	// VERIFIED shape (D2MOO SUNIT_SpawnPresetUnitsInRoom): the three arrive together
	var got []string

	for _, a := range kinds(m.Enter(LevelTristram), d2boss.ActSpawnMonster) {
		got = append(got, a.Key)
	}

	if strings.Join(got, ",") != "ubermephisto,uberbaal,uberdiablo" {
		t.Fatalf("spawns %v", got)
	}

	// killing Mephisto and Diablo is not enough
	for _, c := range []int{ClassUberMephisto, ClassUberDiablo} {
		if as := m.Killed(d2boss.Kill{Class: c, Super: -1}); len(as) != 0 {
			t.Fatalf("early actions %+v", as)
		}
	}

	if e.State() != "tristram" {
		t.Fatalf("state %s", e.State())
	}

	m.Killed(d2boss.Kill{Class: ClassUberBaal, Super: -1})

	var end []d2boss.Action

	for i := 0; i < 100; i++ {
		end = append(end, m.Tick(1)...)
	}

	if d := kinds(end, d2boss.ActDropItem); len(d) != 1 || d[0].Key != CodeStandardOfHeroes {
		t.Fatalf("reward drop %+v", d)
	}

	if q := kinds(end, d2boss.ActPortal); len(q) != 1 || q[0].Level != LevelHarrogath {
		t.Fatalf("exit portal %+v", q)
	}

	if e.State() != "done" {
		t.Fatalf("final state %s", e.State())
	}

	if !strings.Contains(strings.Join(*logs, "\n"), "the event is won") {
		t.Fatal("no win line in the log")
	}
}

func TestFinaleAfterAreaBosses(t *testing.T) {
	m, e, _ := newEvent()

	for _, a := range Areas {
		m.Enter(a.Level)
		m.Killed(d2boss.Kill{Class: a.Boss.Class, Super: -1})
	}

	m.Enter(LevelTristram)
	m.Tick(200)

	for _, c := range []int{ClassUberMephisto, ClassUberDiablo, ClassUberBaal} {
		m.Killed(d2boss.Kill{Class: c, Super: -1})
	}

	m.Tick(100)

	if e.State() != "done" {
		t.Fatalf("the event did not end after the area bosses were killed: %s", e.State())
	}
}

func TestOneKilledIsNotEnough(t *testing.T) {
	m, e, _ := newEvent()
	m.Enter(LevelTristram)
	m.Killed(d2boss.Kill{Class: ClassUberMephisto, Super: -1})
	m.Tick(10)

	if e.State() != "tristram" {
		t.Fatalf("event ended with Diablo and Baal alive: %s", e.State())
	}
}

func TestUseCube(t *testing.T) {
	m, e, _ := newEvent()

	for _, c := range []struct {
		portal string
		want   int
	}{{PortalPandemonium, LevelMatronsDen}, {PortalFinale, LevelTristram}} {
		as, mine := e.UseCube(m, c.portal)
		if !mine || len(as) != 1 || as[0].Level != c.want {
			t.Fatalf("%v: %+v %v", c.portal, as, mine)
		}
	}

	if _, mine := e.UseCube(m, "Cow Portal"); mine {
		t.Fatal("the cow portal does not belong to the event")
	}
}

func TestKeyTreasureClasses(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	raw, err := os.ReadFile(filepath.Join(dir, "itemgen", "patch_d2", "TreasureClassEx.txt"))
	if err != nil {
		t.Skip(err)
	}

	found := map[string]string{}

	for _, line := range strings.Split(string(raw), "\n")[1:] {
		f := strings.Split(line, "\t")
		if len(f) < 2 || strings.HasPrefix(f[0], "Pandemonium Key") {
			continue
		}

		for _, c := range f[1:] {
			if c == CodeKeyTerror || c == CodeKeyHate || c == CodeKeyDestr {
				found[f[0]] = c
			}
		}
	}

	if len(found) != len(KeyTreasureClasses) {
		t.Fatalf("table rows %v, KeyTreasureClasses %v", found, KeyTreasureClasses)
	}

	for tc, code := range KeyTreasureClasses {
		if found[tc] != code {
			t.Errorf("%s: table %q, want %q", tc, found[tc], code)
		}
	}
}
