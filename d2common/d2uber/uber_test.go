package d2uber

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2cube"
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
	tests := []struct {
		name string
		pick func(int) int
		want []int
	}{
		{"default walks the areas in order", nil, []int{133, 134, 135, 135}},
		{"injected choice", func(n int) int { return 2 }, []int{135, 135}},
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

				m.Enter(p[0].Level) // entering marks the area
			}
		})
	}
}

func TestAreaBossesAndOrgans(t *testing.T) {
	tests := []struct {
		level, class int
		key, organ   string
	}{
		{LevelMatronsDen, ClassLilith, "uberandariel", d2cube.CodeHorn},
		{LevelForgottenSands, ClassUberDuriel, "uberduriel", d2cube.CodeBaalEye},
		{LevelFurnaceOfPain, ClassUberIzual, "uberizual", d2cube.CodeMephBrain},
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

	sp := kinds(m.Enter(LevelTristram), d2boss.ActSpawnMonster)
	if len(sp) != 1 || sp[0].Key != "ubermephisto" {
		t.Fatalf("first spawn %+v", sp)
	}

	// Diablo and Baal arrive on their timers
	var got []string

	for i := 0; i < 200; i++ {
		for _, a := range kinds(m.Tick(1), d2boss.ActSpawnMonster) {
			got = append(got, a.Key)
		}
	}

	if strings.Join(got, ",") != "uberdiablo,uberbaal" {
		t.Fatalf("timed spawns %v", got)
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

func TestKilledEarlyBeforeAllArrived(t *testing.T) {
	m, e, _ := newEvent()
	m.Enter(LevelTristram)
	m.Killed(d2boss.Kill{Class: ClassUberMephisto, Super: -1})
	m.Tick(10)

	if e.State() != "tristram" {
		t.Fatalf("event ended with Diablo and Baal still to come: %s", e.State())
	}
}

func TestUseCube(t *testing.T) {
	m, e, _ := newEvent()

	for _, c := range []struct {
		in   []string
		want int
	}{{[]string{"pk1", "pk2", "pk3"}, LevelMatronsDen}, {[]string{"dhn", "bey", "mbr"}, LevelTristram}} {
		r, ok := d2cube.Match(c.in, true)
		if !ok {
			t.Fatalf("no recipe for %v", c.in)
		}

		as, mine := e.UseCube(m, r)
		if !mine || len(as) != 1 || as[0].Level != c.want {
			t.Fatalf("%v: %+v %v", c.in, as, mine)
		}
	}

	r, _ := d2cube.Match([]string{"msf", "vip"}, true)
	if _, mine := e.UseCube(m, r); mine {
		t.Fatal("the staff recipe belongs to the quest system")
	}
}

func TestKeyDrop(t *testing.T) {
	tests := []struct {
		class, diff int
		want        string
	}{
		{156, 2, "pk1"}, {211, 2, "pk2"}, {242, 2, "pk3"}, {156, 1, ""}, {243, 2, ""},
	}

	for _, tt := range tests {
		got, ok := KeyDrop(tt.class, tt.diff)
		if got != tt.want || ok != (tt.want != "") {
			t.Errorf("KeyDrop(%d,%d) = %q,%v", tt.class, tt.diff, got, ok)
		}
	}
}
