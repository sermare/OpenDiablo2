package d2skill

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
)

// Skill level wiring of the missile functions read for the final gaps (Howl
// hit 17, Shout / Battle Cry nova rings with hit 18 / 21, Rabies' bite and
// plague). Numbers are the skills.txt / missiles.txt rows.

func (f *fixture) addSkill(r row) { f.reg.Add(skillFromRow(r)) }
func (f *fixture) addMissile(sp *d2missile.Spec) {
	f.addMissileTo(f.p.Missiles, sp)
	f.addMissileTo(f.sim.Table, sp)
}

func (f *fixture) addMissileTo(t d2missile.Table, sp *d2missile.Spec) {
	t.(missileTable)[strings.ToLower(sp.Name)] = sp
}

func TestHowlMissileGatesAndAppliesTheState(t *testing.T) {
	cases := []struct {
		name  string
		level int // target level; caster 20 + skill level 5 + Param2 1 = 26
		want  string
	}{
		{"weak", 1, "z1:terror:175"},
		{"just below", 25, "z1:terror:175"},
		{"too high", 26, ""},
	}

	for _, c := range cases {
		f := newFixture(map[string]int{"Howl": 5})
		f.u.base = f.u.levels
		f.addMissile(&d2missile.Spec{ID: 148, Name: "howl", SrvDoFunc: 1, SrvHitFunc: 17, Vel: 12, MaxVel: 12, Accel: -1000, Range: 12,
			CollideType: 3, LastCollide: true})
		f.w.targets = []*testTarget{{id: "z1", alive: true, level: c.level, x: 3, y: 0}}

		_, do := f.cast("Howl", 0, 0)
		if !do.OK || len(do.Missiles) != 64 {
			t.Fatalf("%s: %d missiles", c.name, len(do.Missiles))
		}

		for i := 0; i < 15; i++ {
			f.w.frame++
			f.sim.Step()
		}

		got := ""
		if len(f.state) > 0 {
			got = f.state[0]
		}

		if got != c.want {
			t.Errorf("%s: states %v, want %q", c.name, f.state, c.want)
		}
	}
}

func TestShoutAndBattleCryRings(t *testing.T) {
	f := newFixture(map[string]int{"Shout": 5, "Battle Cry": 5})
	f.u.base = f.u.levels
	f.addSkill(row{"skill": "Shout", "Id": "138", "srvdofunc": "68", "srvmissilea": "shout", "aurastate": "shout",
		"auratargetstate": "shout", "auralencalc": "3000", "minmana": "1", "manashift": "8", "mana": "6"})
	f.addSkill(row{"skill": "Battle Cry", "Id": "146", "srvdofunc": "68", "srvmissilea": "battlecry",
		"auratargetstate": "battlecry", "aurafilter": "98304", "auralencalc": "1200", "minmana": "1", "manashift": "8", "mana": "5"})
	f.addMissile(&d2missile.Spec{ID: 149, Name: "shout", SrvDoFunc: 1, SrvHitFunc: 18, Vel: 30, MaxVel: 30, Range: 15,
		CollideType: 3, CollideFriend: true})
	f.addMissile(&d2missile.Spec{ID: 219, Name: "battlecry", SrvDoFunc: 1, SrvHitFunc: 21, Vel: 12, MaxVel: 12, Range: 12,
		CollideType: 3, LastCollide: true, NextHit: true, NextDelay: 4})
	f.w.targets = []*testTarget{{id: "z1", alive: true, level: 1, x: 3, y: 0}}

	// Shout: a ring of 64 plus the timed self state; the world says z1 is an
	// enemy, so the ring buffs nobody
	_, do := f.cast("Shout", 0, 0)
	if !do.OK || len(do.Missiles) != 64 {
		t.Fatalf("shout: %d missiles ok=%v reason=%q", len(do.Missiles), do.OK, do.Reason)
	}

	self := 0

	for _, e := range do.Effects {
		if e.Kind == "self_state" && e.State == "shout" {
			self++
		}
	}

	if self != 1 {
		t.Errorf("shout: %d self states in %+v", self, do.Effects)
	}

	for i := 0; i < 15; i++ {
		f.w.frame++
		f.sim.Step()
	}

	if len(f.state) != 0 {
		t.Errorf("shout buffed an enemy: %v", f.state)
	}

	// Battle Cry: the ring debuffs the enemy for auralencalc frames, once per
	// NextHit window, and the caster gets no state (the skill has no aurastate)
	f.u.cooldowns = map[int]int{}
	f.u.mana = 1000 << 8

	_, do = f.cast("Battle Cry", 0, 0)
	if !do.OK || len(do.Missiles) != 64 {
		t.Fatalf("battle cry: %d missiles ok=%v reason=%q", len(do.Missiles), do.OK, do.Reason)
	}

	for _, e := range do.Effects {
		if e.Kind == "self_state" {
			t.Errorf("battle cry gave the caster a state: %+v", e)
		}
	}

	for i := 0; i < 15; i++ {
		f.w.frame++
		f.sim.Step()
	}

	if len(f.state) == 0 || f.state[0] != "z1:battlecry:1200" {
		t.Errorf("battle cry states %v, want z1:battlecry:1200 first", f.state)
	}
}

// rabiesTarget is a monster that can carry Rabies and own the plague.
type rabiesTarget struct {
	*testTarget
	infected bool
}

func (r *rabiesTarget) HasStateNamed(string) bool { return r.infected }
func (r *rabiesTarget) AsOwner() d2missile.Owner {
	return d2missile.Owner{ID: r.id, Roller: &seq{},
		Pos: func() (float64, float64) { return float64(r.x), float64(r.y) }}
}

func TestRabiesBiteInfectsAndStartsThePlague(t *testing.T) {
	f := newFixture(map[string]int{"Rabies": 5})
	f.u.base = f.u.levels
	f.addSkill(row{"skill": "Rabies", "Id": "238", "srvstfunc": "57", "srvdofunc": "121", "srvmissilea": "rabiesplague",
		"auratargetstate": "rabies", "range": "h2h", "minmana": "1", "manashift": "8", "mana": "10", "ToHit": "50",
		"HitShift": "3", "EType": "pois", "EMin": "6", "EMax": "14", "ELen": "100", "ELevLen1": "10", "ELevLen2": "10",
		"ELevLen3": "10"})
	f.addMissile(&d2missile.Spec{ID: 515, Name: "rabiesplague", SrvDoFunc: 30, Param: [5]int{4, 7}, Range: 10, LevRange: 5,
		CollideType: 3, Size: 3, SubMissile: [3]string{"rabiescontagion"}})
	f.addMissile(&d2missile.Spec{ID: 516, Name: "rabiescontagion", SrvDoFunc: 1, SrvHitFunc: 53, Vel: 5, MaxVel: 5, Range: 20,
		Activate: 2, CollideType: 3, LastCollide: true})

	rt := &rabiesTarget{testTarget: &testTarget{id: "z1", alive: true, level: 1, defense: 10, x: 5, y: 0}}
	tg := Target{Unit: rt, UX: 5, UY: 0, X: 5, Y: 0}

	if st := f.p.Start(f.u, f.id("Rabies"), tg); !st.OK {
		t.Fatalf("start refused: %q", st.Reason)
	}

	do := f.p.Do(f.u, f.id("Rabies"), tg)
	if !do.OK || do.Melee == nil || !do.Melee.Hit {
		t.Fatalf("bite %+v ok=%v reason=%q", do.Melee, do.OK, do.Reason)
	}

	// elemental length: ELen 100 + 10 per level above 1 (tier 1 reaches level 8) = 140 at level 5
	if len(f.state) != 1 || f.state[0] != "z1:rabies:140" {
		t.Errorf("infection states %v, want z1:rabies:140", f.state)
	}

	if len(do.Missiles) != 1 {
		t.Fatalf("%d plague missiles", len(do.Missiles))
	}

	pm := do.Missiles[0]
	if pm.Spec.Name != "rabiesplague" || pm.Owner.ID != "z1" || pm.MarkOwner == nil || pm.MarkOwner.ID != "hero" {
		t.Errorf("plague %v owner %q mark %+v: it must belong to the infected monster and mark the caster", pm, pm.Owner.ID, pm.MarkOwner)
	}

	if pm.Damage != (d2missile.DamageDesc{}) {
		t.Errorf("the plague itself carries damage %+v", pm.Damage)
	}

	// an already infected monster is bitten (damage) but not infected again
	f.state = nil
	rt.infected = true
	f.u.cooldowns = map[int]int{}
	f.u.mana = 1000 << 8

	if st := f.p.Start(f.u, f.id("Rabies"), tg); !st.OK {
		t.Fatalf("second start refused: %q", st.Reason)
	}

	do = f.p.Do(f.u, f.id("Rabies"), tg)
	if !do.OK || len(do.Missiles) != 0 || len(f.state) != 0 {
		t.Errorf("second bite: %d missiles, states %v", len(do.Missiles), f.state)
	}
}
