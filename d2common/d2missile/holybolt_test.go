package d2missile

import "testing"

type kindTarget struct {
	fakeTarget
	undead, demon bool
	healed        int
}

func (k *kindTarget) IsUndead() bool { return k.undead }
func (k *kindTarget) IsDemon() bool  { return k.demon }
func (k *kindTarget) Heal(a int)     { k.healed += a }

type kindWorld struct {
	*fakeWorld
	t *kindTarget
}

func (w *kindWorld) Targets(x, y int) []Target {
	if w.t.alive && w.t.x == x && w.t.y == y {
		return []Target{w.t}
	}

	return nil
}

func (w *kindWorld) IsEnemy(_ Owner, t Target) bool { return !t.(*kindTarget).friendly }

func holyBolt(par1, par2 int) *Spec {
	return &Spec{ID: 55, Name: "holybolt", SrvDoFunc: 1, SrvHitFunc: 7, Vel: 20, MaxVel: 20, Range: 50,
		CollideType: 3, CollideKill: true, CollideFriend: true, LastCollide: true,
		SHitPar: [3]int{par1, par2}}
}

func holyRun(sp *Spec, tg *kindTarget) (map[EventKind]int, int) {
	w := &kindWorld{fakeWorld: newWorld(), t: tg}
	s := NewSim(w, nil)
	o := Owner{ID: "hero", IsPlayer: true, Roller: fakeRoller{0}}
	_, _ = s.Create(CreateParams{Spec: sp, Level: 1, DestX: 60, Owner: o, HealMin: 5 << 8, HealMax: 9 << 8,
		Damage: DamageDesc{PhysMin: 256, PhysMax: 256}})

	var evs []Event

	s.OnEvent = func(e Event) { evs = append(evs, e) }

	for i := 0; i < 60; i++ {
		w.frame++
		s.Step()
	}

	return kinds(evs), len(s.Missiles())
}

// Hit function 7 (0x5a7a40, verified): allies are healed (no damage, missile
// ends), enemies are filtered by sHitPar2 (0 all, 1 undead, 2 demons); a
// filtered unit lets the bolt fly on.
func TestHolyBolt(t *testing.T) {
	tg := &kindTarget{fakeTarget: *newTarget("ally", 10, 0)}
	tg.friendly = true

	k, left := holyRun(holyBolt(1, 1), tg)
	if k[EventHeal] != 1 || k[EventHit] != 0 || tg.healed != 5<<8 || left != 0 {
		t.Errorf("ally: events %v healed %d left %d", k, tg.healed, left)
	}

	for _, tc := range []struct {
		name          string
		par2          int
		undead, demon bool
		hits          int
	}{
		{"all", 0, false, false, 1},
		{"undead filter, undead", 1, true, false, 1},
		{"undead filter, living", 1, false, true, 0},
		{"demon filter, demon", 2, false, true, 1},
		{"demon filter, living", 2, true, false, 0},
	} {
		e := &kindTarget{fakeTarget: *newTarget("e", 10, 0), undead: tc.undead, demon: tc.demon}
		k, left := holyRun(holyBolt(1, tc.par2), e)

		if k[EventHit] != tc.hits || e.healed != 0 || (tc.hits == 0) != (k[EventExpire] == 1) || left != 0 {
			t.Errorf("%s: events %v left %d", tc.name, k, left)
		}
	}
}
