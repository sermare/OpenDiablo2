package d2skills

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2missile"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

type castCall struct {
	u            d2skill.Unit
	skillID      int
	missile      string
	fromX, fromY int
	tgt          d2skill.Target
}

type fakeCaster struct {
	calls []castCall
	fire  bool
}

func (f *fakeCaster) CastTrap(u d2skill.Unit, id int, missile string, fx, fy int, tg d2skill.Target) *d2missile.Missile {
	f.calls = append(f.calls, castCall{u, id, missile, fx, fy, tg})

	if !f.fire {
		return nil
	}

	return &d2missile.Missile{}
}

func trapTestEngine(f *fakeCaster) *Engine {
	return &Engine{Logger: d2util.NewLogger(), fakeCaster: f, targets: map[string]*monsterTarget{},
		heroes: map[string]*heroUnit{}}
}

func monAt(x, y int) *d2mapentity.Monster {
	m := &d2mapentity.Monster{}
	m.Position.Set(float64(x), float64(y))

	return m
}

// The old driver (trapsTick -> fireTrapAt) and the Director callback
// (FireTrap) must hand the pipeline the same skill, missile, start cell,
// caster and target.
func TestTrapFireParity(t *testing.T) {
	owner := &d2mapentity.Player{}
	foe := monAt(14, 9)
	trap := monAt(10, 10)

	fo := &fakeCaster{fire: true}
	eo := trapTestEngine(fo)
	run := &trapRun{m: trap, u: eo.hero(owner), skillID: 271, missile: "sentry shot", skillName: "Lightning Sentry"}

	if !eo.fireTrapAt(eo.trapCaster(), run, 10, 10, foe) {
		t.Fatal("old path did not fire")
	}

	fn := &fakeCaster{fire: true}
	en := trapTestEngine(fn)
	en.heroes[owner.ID()] = eo.heroes[owner.ID()] // same hero unit: same skill levels and mana

	ok := en.FireTrap(d2monsters.TrapShot{Trap: trap, Owner: owner, FromX: 10, FromY: 10, Target: foe,
		Spec: d2monsters.TrapSpec{SkillID: 271, SkillName: "Lightning Sentry", Missile: "sentry shot"}})
	if !ok {
		t.Fatal("new path did not fire")
	}

	if len(fo.calls) != 1 || len(fn.calls) != 1 {
		t.Fatalf("calls old=%d new=%d", len(fo.calls), len(fn.calls))
	}

	a, b := fo.calls[0], fn.calls[0]

	// the monster target adapter is per engine: compare it by monster
	if a.tgt.Unit.ID() != b.tgt.Unit.ID() {
		t.Error("target unit differs")
	}

	a.tgt.Unit, b.tgt.Unit = nil, nil

	if a.u != b.u || !reflect.DeepEqual(a.tgt, b.tgt) || a.skillID != b.skillID || a.missile != b.missile ||
		a.fromX != b.fromX || a.fromY != b.fromY {
		t.Errorf("old %+v != new %+v", a, b)
	}

	// a trap shot spends no mana and starts no cooldown in either path
	if h := eo.hero(owner); len(h.cooldowns) != 0 {
		t.Errorf("cooldowns %v", h.cooldowns)
	}
}

func TestFireTrapGuards(t *testing.T) {
	f := &fakeCaster{}
	e := trapTestEngine(f)

	if e.FireTrap(d2monsters.TrapShot{Trap: monAt(0, 0), Target: monAt(1, 1)}) || len(f.calls) != 0 {
		t.Error("no owner must not fire")
	}

	if e.FireTrap(d2monsters.TrapShot{Trap: monAt(0, 0), Owner: &d2mapentity.Player{}}) || len(f.calls) != 0 {
		t.Error("no target must not fire")
	}

	if e.FireTrap(d2monsters.TrapShot{Trap: monAt(0, 0), Owner: &d2mapentity.Player{}, Target: monAt(1, 1)}) {
		t.Error("a pipeline miss must report false")
	}
}
