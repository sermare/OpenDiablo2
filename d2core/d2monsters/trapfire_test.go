package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

type fakeFirer struct {
	shots []TrapShot
	ok    bool
}

func (f *fakeFirer) FireTrap(s TrapShot) bool {
	f.shots = append(f.shots, s)

	return f.ok
}

func TestCastRoutesToSkillFirer(t *testing.T) {
	sentry := petUnit(1, 10, 10, "AssassinSentry", "trap")
	foe := petUnit(2, 14, 10, "Melee", "")
	dead := petUnit(3, 12, 10, "Melee", "")
	dead.m.Vitals.HP = 0

	d := testDirector(sentry, foe, dead)
	owner := &d2mapentity.Player{}
	sentry.ally.owner = owner
	spec := TrapSpec{SkillID: 271, SkillName: "Lightning Sentry", Missile: "sentry shot"}
	sentry.ally.trap = &spec

	// no firer injected: not handled, the monstats cast runs as before
	if handled, _ := d.fireTrap(sentry, d2monster.Target{ID: 2}); handled {
		t.Fatal("handled without a firer")
	}

	f := &fakeFirer{ok: true}
	d.SetSkillFirer(f)

	if handled, ok := d.fireTrap(sentry, d2monster.Target{ID: 2}); !handled || !ok {
		t.Fatalf("handled=%v ok=%v", handled, ok)
	}

	if len(f.shots) != 1 {
		t.Fatalf("shots %d", len(f.shots))
	}

	s := f.shots[0]
	if s.Spec != spec || s.Owner != owner || s.Trap != sentry.m || s.Target != foe.m || s.FromX != 10 || s.FromY != 10 {
		t.Errorf("shot %+v", s)
	}

	// a vanished target is handled (no melee fallback) but does not fire
	if handled, ok := d.fireTrap(sentry, d2monster.Target{ID: 99}); !handled || ok || len(f.shots) != 1 {
		t.Errorf("unknown target: handled=%v ok=%v shots=%d", handled, ok, len(f.shots))
	}

	// unarmed units and hostile monsters are never routed
	sentry.ally.trap = nil

	if handled, _ := d.fireTrap(sentry, d2monster.Target{ID: 2}); handled {
		t.Error("unarmed sentry routed")
	}

	if handled, _ := d.fireTrap(foe, d2monster.Target{ID: 1}); handled {
		t.Error("hostile monster routed")
	}

	if d.firer == nil {
		t.Error("firer lost")
	}

	d.SetSkillFirer(nil)
	sentry.ally.trap = &spec

	if handled, _ := d.fireTrap(sentry, d2monster.Target{ID: 2}); handled {
		t.Error("handled after the firer was removed")
	}
}

func TestArmTrap(t *testing.T) {
	sentry := petUnit(1, 0, 0, "AssassinSentry", "trap")
	foe := petUnit(2, 5, 5, "Melee", "")
	d := testDirector(sentry, foe)
	d.byEntity[sentry.m.ID()] = sentry

	if !d.ArmTrap(sentry.m, TrapSpec{SkillID: 7}) || sentry.ally.trap == nil || sentry.ally.trap.SkillID != 7 {
		t.Error("ally not armed")
	}

	d.byEntity[foe.m.ID()] = foe // zero-value monsters share the id "": the hostile replaces the sentry

	if d.ArmTrap(foe.m, TrapSpec{}) {
		t.Error("a hostile unit cannot be armed")
	}
}

// The AI side: thinkSentry's Cast reaches the firer for an armed sentry whose
// target is in range, through the real Director.Cast path.
func TestSentryThinkFiresSkill(t *testing.T) {
	sentry := petUnit(1, 10, 10, "AssassinSentry", "trap")
	foe := petUnit(2, 12, 10, "Melee", "")
	d := testDirector(sentry, foe)
	f := &fakeFirer{ok: true}
	d.SetSkillFirer(f)
	sentry.ally.trap = &TrapSpec{SkillID: 1, Missile: "m"}
	sentry.b.Profile = &d2monster.Profile{}

	tg, dist, ok := d.AttackTarget(sentry.b)
	if !ok || tg.ID != 2 {
		t.Fatalf("attack target %+v ok=%v", tg, ok)
	}

	if dist > meleeInRange {
		t.Fatalf("test setup: dist %d", dist)
	}

	if !d.Cast(sentry.b, 0, tg) || len(f.shots) != 1 || f.shots[0].Target != foe.m {
		t.Errorf("Cast did not reach the firer: %d shots", len(f.shots))
	}
}
