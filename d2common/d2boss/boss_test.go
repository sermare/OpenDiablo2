package d2boss

import (
	"strings"
	"testing"
)

func newTest() (*Manager, *[]string) {
	var log []string

	m := New(func(s string) { log = append(log, s) })

	return m, &log
}

func kinds(as []Action) string {
	var out []string
	for _, a := range as {
		out = append(out, a.Kind.String()+":"+a.Name)
	}

	return strings.Join(out, ",")
}

func TestDurielTomb(t *testing.T) {
	m, _ := newTest()
	tomb := m.Encounter("duriel").(*Tomb)

	// the orifice without the staff does nothing
	if as := m.Operate(Operate{Object: ObjOrifice}); len(as) != 0 || tomb.State() != "sealed" {
		t.Fatalf("orifice without the staff: %v %s", as, tomb.State())
	}

	// with the staff: the orifice reacts and the portal opens after the delay
	as := m.Operate(Operate{Object: ObjOrifice, HasStaff: true})
	if kinds(as) != "object-mode:orifice" || tomb.State() != "staff-placed" {
		t.Fatalf("staff: %v %s", kinds(as), tomb.State())
	}

	if as = m.Tick(tombPortalDelay - 1); len(as) != 0 {
		t.Fatalf("portal too early: %v", as)
	}

	as = m.Tick(1)
	if len(as) != 1 || as[0].Kind != ActPortal || as[0].Level != LevelDurielLair || tomb.State() != "portal-open" {
		t.Fatalf("portal: %v %s", as, tomb.State())
	}

	// entering the lair spawns Duriel once
	as = m.Enter(LevelDurielLair)
	if len(as) != 1 || as[0].Class != ClassDuriel || !as[0].Boss || tomb.State() != "in-lair" {
		t.Fatalf("lair: %v", as)
	}

	if as = m.Enter(LevelDurielLair); len(as) != 0 {
		t.Fatalf("a second Duriel: %v", as)
	}

	// his death: message, orifice, Tyrael after the timer
	as = m.Killed(Kill{Class: ClassDuriel, Super: -1})
	if kinds(as) != "message:Duriel dead,object-mode:orifice" || as[0].Msg != 0x3d || tomb.State() != "done" {
		t.Fatalf("kill: %v", kinds(as))
	}

	as = m.Tick(tyraelDelay)
	if kinds(as) != "spawn-npc:Tyrael,portal:portal to Lut Gholein" {
		t.Fatalf("after the kill: %v", kinds(as))
	}
}

func TestMephistoWakes(t *testing.T) {
	m, _ := newTest()
	me := m.Encounter("mephisto").(*Mephisto)
	me.X, me.Y = 100, 100

	m.HeroX, m.HeroY = 300, 100
	as := m.Enter(LevelDurance3)

	if len(as) != 1 || !as[0].Asleep || as[0].Class != ClassMephisto || me.State() != "asleep" {
		t.Fatalf("enter: %v", as)
	}

	if as = m.Tick(10); len(as) != 0 {
		t.Fatalf("woke too early: %v", as)
	}

	m.HeroX = 120
	as = m.Tick(1)

	if len(as) != 1 || as[0].Kind != ActWake || me.State() != "awake" {
		t.Fatalf("wake: %v %s", as, me.State())
	}

	as = m.Killed(Kill{Class: ClassMephisto, Super: -1})
	if len(as) != 1 || as[0].Msg != 0x3e || me.State() != "dead" {
		t.Fatalf("kill: %v", as)
	}

	as = m.Tick(hellgateDelay)
	if len(as) != 1 || as[0].Kind != ActPortal || as[0].Class != ObjHellgate || as[0].Level != LevelFortress {
		t.Fatalf("hellgate: %v", as)
	}
}

func TestMephistoBridgeWakes(t *testing.T) {
	m, _ := newTest()
	m.HeroX, m.HeroY = 900, 900
	m.Enter(LevelDurance3)

	as := m.Operate(Operate{Object: ObjMephistoBridge, QuestOpen: true})
	if len(as) != 1 || as[0].Kind != ActWake {
		t.Fatalf("bridge: %v", as)
	}
}

func TestSealsAndDiablo(t *testing.T) {
	m, _ := newTest()
	d := m.Encounter("diablo").(*Seals)

	// a plain seal opens without a boss
	if as := m.Operate(Operate{Object: ObjSealPlainA}); len(as) != 0 {
		t.Fatalf("plain seal: %v", as)
	}

	// the same seal twice does nothing
	m.Operate(Operate{Object: ObjSealPlainA})

	if d.count() != 1 {
		t.Fatalf("seals open %d", d.count())
	}

	// the boss seals spawn their super uniques with the verified groups
	want := map[int][3]int{
		ObjSealVizier:   {SuperVizier, ClassVizier, 9},
		ObjSealDeSeis:   {SuperDeSeis, ClassDeSeis, 5},
		ObjSealInfector: {SuperInfector, ClassInfector, 9},
	}

	for _, obj := range []int{ObjSealVizier, ObjSealDeSeis, ObjSealInfector} {
		as := m.Operate(Operate{Object: obj, X: 10, Y: 20})
		w := want[obj]

		if len(as) != 1 || as[0].Super != w[0] || as[0].Class != w[1] || as[0].Group != w[2] || as[0].X != 10 {
			t.Fatalf("seal %d: %v", obj, as)
		}
	}

	m.Operate(Operate{Object: ObjSealPlainB})

	if d.count() != 5 {
		t.Fatalf("seals open %d", d.count())
	}

	// Diablo needs the three seal bosses dead too
	for i, su := range []int{SuperVizier, SuperDeSeis, SuperInfector} {
		as := m.Killed(Kill{Class: want[map[int]int{SuperVizier: ObjSealVizier, SuperDeSeis: ObjSealDeSeis, SuperInfector: ObjSealInfector}[su]][1], Super: su})

		if i < 2 && len(as) != 0 {
			t.Fatalf("Diablo came early: %v", as)
		}

		if i == 2 && len(as) != 1 {
			t.Fatalf("no summons message after the third boss: %v", as)
		}
	}

	if d.State() != "arriving" {
		t.Fatalf("state %s", d.State())
	}

	as := m.Tick(diabloDelay)
	if len(as) != 1 || as[0].Class != ClassDiablo || d.State() != "diablo-alive" {
		t.Fatalf("Diablo: %v %s", as, d.State())
	}

	// a repeated trigger does nothing (one-shot)
	if as = m.Killed(Kill{Class: ClassVizier, Super: SuperVizier}); len(as) != 0 {
		t.Fatalf("arrival twice: %v", as)
	}

	m.Killed(Kill{Class: ClassDiablo, Super: -1})

	if d.State() != "diablo-dead" {
		t.Fatalf("state %s", d.State())
	}
}

func TestSealOrderDoesNotMatter(t *testing.T) {
	// the bosses die before the last plain seal is opened: Diablo comes with the last seal
	m, _ := newTest()
	for _, o := range []int{ObjSealVizier, ObjSealDeSeis, ObjSealInfector} {
		m.Operate(Operate{Object: o})
	}

	m.Killed(Kill{Class: ClassVizier, Super: SuperVizier})
	m.Killed(Kill{Class: ClassDeSeis, Super: SuperDeSeis})
	m.Killed(Kill{Class: ClassInfector, Super: SuperInfector})
	m.Operate(Operate{Object: ObjSealPlainA})

	if as := m.Operate(Operate{Object: ObjSealPlainB}); len(as) != 1 {
		t.Fatalf("summons expected with the fifth seal: %v", as)
	}
}

func TestThroneWaves(t *testing.T) {
	m, _ := newTest()
	th := m.Encounter("baal").(*Throne)

	as := m.Enter(LevelThrone)
	if len(as) != 1 || as[0].Key != "baalthrone" {
		t.Fatalf("throne: %v", as)
	}

	groups := []int{5, 3, 5, 8, 5}

	for w := 0; w < 5; w++ {
		if _, ok := m.ThroneStep(StepAnnounce, w); !ok {
			t.Fatal("announce failed")
		}

		as, ok := m.ThroneStep(StepSpawn, w)
		if !ok || len(as) != 1 || as[0].Super != 62+w || as[0].Group != groups[w] {
			t.Fatalf("wave %d: %v", w, as)
		}

		if m.WaveCleared() {
			t.Fatalf("wave %d cleared before a kill", w)
		}

		// kill the whole wave
		for i := 0; i < 1+groups[w]; i++ {
			m.Killed(Kill{Class: as[0].Class, Super: -1, Level: LevelThrone})
		}

		if !m.WaveCleared() {
			t.Fatalf("wave %d not cleared: %s", w, th.State())
		}
	}

	if _, ok := m.ThroneStep(StepSpawn, 5); ok {
		t.Fatal("a sixth wave")
	}

	// morph: Baal to the stairs, the portal appears, Baal enters it
	as, _ = m.ThroneStep(StepMorph, 5)
	if len(as) != 1 || as[0].Key != "baalcrabtostairs" {
		t.Fatalf("morph: %v", as)
	}

	as = m.Tick(baalPortalDelay)
	if len(as) != 1 || as[0].Class != ObjWorldstone {
		t.Fatalf("portal object: %v", as)
	}

	as = m.BaalLeft()
	if len(as) != 1 || as[0].Kind != ActPortal || as[0].Level != LevelWorldstone {
		t.Fatalf("left: %v", as)
	}

	as = m.Enter(LevelWorldstone)
	if len(as) != 1 || as[0].Key != "baalcrab" {
		t.Fatalf("chamber: %v", as)
	}

	m.Killed(Kill{Class: 544, Super: -1, Level: LevelWorldstone})

	if !strings.HasPrefix(th.State(), "baal-dead") {
		t.Fatalf("state %s", th.State())
	}
}

func TestStatesString(t *testing.T) {
	m, _ := newTest()
	if s := m.States(); !strings.Contains(s, "duriel=sealed") || !strings.Contains(s, "mephisto=idle") {
		t.Fatal(s)
	}
}
