package d2gamescreen

import (
	"reflect"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero/herogen"
)

func TestLeftFightPick(t *testing.T) {
	sv := func(name, rng string, pts int, supported bool) skillView {
		return skillView{ID: 40, Name: name, Points: pts, Range: rng, Supported: supported}
	}

	tests := []struct {
		name    string
		s       skillView
		dropped map[int]bool
		ok      bool
		melee   bool
	}{
		{"necromancer Bone Spear", sv("Bone Spear", "none", 20, true), nil, true, false},
		{"amazon Lightning Fury", sv("Lightning Fury", "rng", 20, true), nil, true, false},
		{"amazon Guided Arrow", sv("Guided Arrow", "rng", 20, true), nil, true, false},
		{"druid Firestorm", sv("Firestorm", "none", 20, true), nil, true, false},
		{"sorceress Fire Ball", sv("Fire Ball", "none", 20, true), nil, true, false},
		{"paladin Zeal", sv("Zeal", "h2h", 20, true), nil, true, true},
		{"paladin Smite", sv("Smite", "h2h", 1, true), nil, true, true},
		{"barbarian Frenzy", sv("Frenzy", "h2h", 20, true), nil, true, true},
		{"assassin Dragon Talon", sv("Dragon Talon", "h2h", 20, true), nil, true, true},
		{"no points", sv("Bone Spear", "none", 0, true), nil, false, false},
		{"engine cannot cast it", sv("Bone Spear", "none", 20, false), nil, false, false},
		{"dropped in this fight", sv("Bone Spear", "none", 20, true), map[int]bool{40: true}, false, false},
		{"a buff on the left button", sv("Bone Armor", "none", 20, true), nil, false, false},
		{"a passive", sv("Claw Mastery", "none", 20, true), nil, false, false},
		{"an aura", sv("Fanaticism", "none", 20, true), nil, false, false},
	}

	for _, tt := range tests {
		p, ok := leftFightPick(tt.s, tt.dropped)
		if ok != tt.ok {
			t.Errorf("%s: ok = %v, want %v", tt.name, ok, tt.ok)
			continue
		}

		if ok && (p.Melee != tt.melee || !p.Left || p.ID != tt.s.ID) {
			t.Errorf("%s: pick %+v, want melee %v", tt.name, p, tt.melee)
		}
	}
}

func TestPlanSupports(t *testing.T) {
	sv := func(id int, name string, pts int, supported bool) skillView {
		return skillView{ID: id, Name: name, Points: pts, Supported: supported}
	}

	right := sv(68, "Bone Armor", 15, true)
	hot := []skillView{
		sv(70, "Raise Skeleton", 15, true),
		sv(75, "Clay Golem", 1, true),
		sv(98, "Might", 9, true),       // an aura on a hotkey is not cast: it would replace the right button's
		sv(84, "Bone Spear", 20, true), // an attack
		sv(68, "Bone Armor", 15, true), // the right button again
		sv(94, "FireGolem", 0, true),   // no points
		sv(95, "Revive", 1, false),     // the engine cannot cast it (and it is no support of the table)
		sv(80, "Raise Skeletal Mage", 1, false),
	}

	got := planSupports(right, hot)

	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}

	if want := []string{"Bone Armor", "Raise Skeleton", "Clay Golem"}; !reflect.DeepEqual(names, want) {
		t.Errorf("supports %v, want %v", names, want)
	}

	if got[0].Every != supportSeconds[roleBuff] || got[1].Every != supportSeconds[roleSummon] {
		t.Errorf("intervals %+v", got)
	}

	// a paladin: the aura of the right button is cast, once
	pal := planSupports(sv(122, "Fanaticism", 20, true), []skillView{sv(99, "Prayer", 1, true), sv(98, "Might", 9, true)})
	if len(pal) != 1 || pal[0].Name != "Fanaticism" || pal[0].Every != supportSeconds[roleAura] {
		t.Errorf("paladin supports %+v", pal)
	}

	// the plain attack on the right button is no support
	if got := planSupports(sv(0, "Attack", 1, true), nil); len(got) != 0 {
		t.Errorf("supports of nothing: %+v", got)
	}
}

func TestDueSupport(t *testing.T) {
	plan := []fightSupport{{ID: 1, Name: "a", Every: 20}, {ID: 2, Name: "b", Every: 45}}
	last := map[int]float64{}

	s, ok := dueSupport(plan, last, 100)
	if !ok || s.ID != 1 {
		t.Fatalf("first due = %+v %v", s, ok)
	}

	last[1] = 100

	if s, ok = dueSupport(plan, last, 101); !ok || s.ID != 2 {
		t.Errorf("after a: %+v %v", s, ok)
	}

	last[2] = 101

	if _, ok = dueSupport(plan, last, 119.9); ok {
		t.Error("nothing is due 19.9 s after a")
	}

	if s, ok = dueSupport(plan, last, 120); !ok || s.ID != 1 {
		t.Errorf("a is due again at 120: %+v %v", s, ok)
	}

	// time zero is a time, not "never cast"
	if _, ok = dueSupport(plan, map[int]float64{1: 0, 2: 0}, 5); ok {
		t.Error("a skill cast at clock 0 is due again after 5 s")
	}
}

func TestRefusalCounts(t *testing.T) {
	for reason, want := range map[string]bool{
		d2skill.ReasonMana: false, d2skill.ReasonCooldown: false, d2skill.ReasonLOS: true, d2skill.ReasonAmmo: true,
		d2skill.ReasonTarget: true, d2skill.ReasonMissile: true, "": true,
	} {
		if refusalCounts(reason) != want {
			t.Errorf("refusalCounts(%q) = %v, want %v", reason, !want, want)
		}
	}
}

func TestIsPlainAttack(t *testing.T) {
	for id, want := range map[int]bool{0: true, 5: true, 6: false, 36: false, 255: false} {
		if isPlainAttack(id) != want {
			t.Errorf("isPlainAttack(%d) = %v", id, !want)
		}
	}
}

// TestPresetSkillsAreKnownToTheFight proves every generated hero's left skill is an attack of the fight table
// and every right button skill, or hotkey, that is a support is known to it by name, so the fight uses the
// build the preset gives the class.
func TestPresetSkillsAreKnownToTheFight(t *testing.T) {
	for _, n := range herogen.PresetNames() {
		c, _ := herogen.ClassOfPreset(n)

		spec, err := herogen.Preset(c, herogen.DefaultName(c))
		if err != nil {
			t.Fatal(err)
		}

		left := herogen.SkillName(c, spec.Left)
		if !fightAttackSkills[left] {
			t.Errorf("%s: the left skill %q is no attack of the fight table", n, left)
		}

		if fightSupportSkills[left] != roleNone {
			t.Errorf("%s: the left skill %q is both an attack and a support", n, left)
		}

		right := herogen.SkillName(c, spec.Right)
		if !fightAttackSkills[right] && fightSupportSkills[right] == roleNone {
			t.Errorf("%s: the right skill %q is neither an attack nor a support of the fight table", n, right)
		}

		supports := 0

		for _, id := range append([]int{spec.Right}, spec.Hotkeys...) {
			if fightSupportSkills[herogen.SkillName(c, id)] != roleNone {
				supports++
			}
		}

		if supports < 1 {
			t.Errorf("%s: no right button or hotkey skill is a support of the fight", n)
		}

		if spec.Hotkeys[0] != spec.Left {
			t.Errorf("%s: F1 is skill %d, the left skill %d", n, spec.Hotkeys[0], spec.Left)
		}
	}

	// the names of the tables are skill names of the seven classes
	known := map[string]bool{}

	for c := d2s.Amazon; c <= d2s.Assassin; c++ {
		for _, name := range herogen.ClassSkillNames(c) {
			known[name] = true
		}
	}

	for name := range fightAttackSkills {
		if !known[name] {
			t.Errorf("attack skill %q is no player skill name", name)
		}
	}

	for name := range fightSupportSkills {
		if !known[name] {
			t.Errorf("support skill %q is no player skill name", name)
		}

		if fightAttackSkills[name] {
			t.Errorf("%q is in both tables", name)
		}
	}
}
