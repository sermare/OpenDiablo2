package d2monsters

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// The numbers below are read off the 1.14b patch_d2 skills.txt / missiles.txt
// (columns quoted per case) and the expected damage is worked out by hand from
// the verified level-tier rule (levels 2..8 add Lev1 each) and HitShift.

func missileRow(hs, min, max, minLev, maxLev int, etype string, emin, emax, eminLev, emaxLev, elen int) *d2records.MissileRecord {
	m := &d2records.MissileRecord{HitShift: hs}
	m.Damage.MinDamage, m.Damage.MaxDamage = min, max
	m.Damage.MinLevelDamage[0], m.Damage.MaxLevelDamage[0] = minLev, maxLev
	m.ElementalDamage.ElementType = etype
	m.ElementalDamage.Damage.MinDamage, m.ElementalDamage.Damage.MaxDamage = emin, emax
	m.ElementalDamage.Damage.MinLevelDamage[0], m.ElementalDamage.Damage.MaxLevelDamage[0] = eminLev, emaxLev
	m.ElementalDamage.Duration = elen

	return m
}

func TestSkillDamageTable(t *testing.T) {
	base := d2mapentity.MonsterAttack{ToHit: 100, Min: 10, Max: 20}

	iceSpear := &d2records.SkillRecord{Srvdofunc: 2, HitShift: 8, SrcDam: 128, EType: "cold",
		EMin: 10, EMax: 14, EMinLev1: 8, EMaxLev1: 9}
	baalNova := &d2records.SkillRecord{Srvdofunc: 22, HitShift: 8, EType: "fire", EMin: 50, EMax: 75, EMinLev1: 24, EMaxLev1: 24}
	fireBall := &d2records.SkillRecord{Skill: "Fire Ball", Srvmissile: "fireball", HitShift: 7, EType: "fire",
		EMin: 12, EMax: 28, EMinLev1: 13, EMaxLev1: 15}
	curse := &d2records.SkillRecord{Srvdofunc: 30}
	attack := &d2records.SkillRecord{Srvdofunc: 1, HitShift: 0, SrcDam: 128}
	half := &d2records.SkillRecord{Srvdofunc: 1, SrcDam: 64}

	tests := []struct {
		name string
		rec  *d2records.SkillRecord
		lvl  int
		mr   *d2records.MissileRecord
		lr   *d2records.SkillRecord
		want d2mapentity.MonsterAttack
		ok   bool
	}{
		{"Attack uses 100 percent of the mode damage", attack, 1, nil, nil,
			d2mapentity.MonsterAttack{ToHit: 100, Min: 10, Max: 20}, true},
		{"SrcDam 64 halves it", half, 1, nil, nil, d2mapentity.MonsterAttack{ToHit: 100, Min: 5, Max: 10}, true},
		// MonIceSpear lvl 5: cold 10+4*8=42 .. 14+4*9=50 on top of the unit damage
		{"MonIceSpear lvl5", iceSpear, 5, nil, nil,
			d2mapentity.MonsterAttack{ToHit: 100, Min: 10, Max: 20, ElemType: "cold", ElemMin: 42, ElemMax: 50}, true},
		// Baal Nova lvl 1: own fire 50..75, SrcDam 0 so no unit damage
		{"Baal Nova lvl1", baalNova, 1, nil, nil,
			d2mapentity.MonsterAttack{ToHit: 100, ElemType: "fire", ElemMin: 50, ElemMax: 75}, true},
		// Siege Beast Stomp lvl 3: 20+2*10 .. 60+2*MaxLevDam1(0)
		{"Stomp physical columns", &d2records.SkillRecord{Srvdofunc: 134, HitShift: 8, MinDam: 20, MaxDam: 60, MinLevDam1: 10},
			3, nil, nil, d2mapentity.MonsterAttack{ToHit: 100, Min: 40, Max: 60}, true},
		// summoner Fire Ball lvl 5: (12+4*13)<<7>>8 = 32 .. (28+4*15)<<7>>8 = 44, missile is Skill-linked
		{"Fire Ball linked missile lvl5", fireBall, 5, &d2records.MissileRecord{SkillName: "Fire Ball"}, fireBall,
			d2mapentity.MonsterAttack{ToHit: 100, ElemType: "fire", ElemMin: 32, ElemMax: 44}, true},
		// Mephisto frost bolt lvl 5: phys 12+4*8=44 .. 16+4*8=48, cold 12+4*18=84 .. 18+4*20=98
		{"mephisto missile own columns", &d2records.SkillRecord{Srvmissile: "mephisto"}, 5,
			missileRow(8, 12, 16, 8, 8, "cold", 12, 18, 18, 20, 75), nil,
			d2mapentity.MonsterAttack{ToHit: 100, Min: 44, Max: 48, ElemType: "cold", ElemMin: 84, ElemMax: 98}, true},
		// Andariel poison bolt lvl 1: raw 8.8 physical 1280..1792 = 5..7, poison rate 32..64 for 800 frames = 100..200
		{"andypoisonbolt poison total", &d2records.SkillRecord{Srvmissile: "andypoisonbolt"}, 1,
			missileRow(0, 1280, 1792, 1280, 1280, "pois", 32, 64, 38, 38, 800), nil,
			d2mapentity.MonsterAttack{ToHit: 100, Min: 5, Max: 7, ElemType: "pois", ElemMin: 100, ElemMax: 200}, true},
		// plain-missile with SrcDamage 128 and no own columns: the unit damage flies
		{"cr_arrow SrcDamage", &d2records.SkillRecord{Srvmissile: "cr_arrow6"}, 1,
			&d2records.MissileRecord{SourceDamage: 128}, nil, d2mapentity.MonsterAttack{ToHit: 100, Min: 10, Max: 20}, true},
		{"curse deals nothing", curse, 3, nil, nil, d2mapentity.MonsterAttack{}, false},
	}

	for _, tt := range tests {
		sk := &monSkill{rec: tt.rec, level: tt.lvl, missile: tt.rec.Srvmissile,
			kind: d2monster.ClassifyEffect(tt.rec.Srvdofunc, tt.rec.Srvmissile != "")}

		got, ok := skillDamage(base, sk, tt.mr, tt.lr)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("%s: got %+v ok=%v want %+v ok=%v", tt.name, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSkillSlotFor(t *testing.T) {
	var p d2monster.Profile

	p.Skills[0] = d2monster.SkillSlot{Name: "A", Mode: d2monster.ModeSkill1}
	p.Skills[2] = d2monster.SkillSlot{Name: "C", Mode: d2monster.ModeSkill2}
	p.Skills[3] = d2monster.SkillSlot{Name: "D", Mode: d2monster.ModeCast}

	tests := []struct {
		mode d2monster.Mode
		slot int
		want int
	}{
		{d2monster.ModeSkill2, -1, 2},   // AI names only the mode: first slot in that mode
		{d2monster.ModeSkill2, 0, 0},    // explicit slot wins
		{d2monster.ModeAttack1, -1, -1}, // plain attack has no skill
		{d2monster.ModeAttack2, -1, -1}, //
		{d2monster.ModeSkill3, -1, -1},  // no slot in that mode
		{d2monster.ModeCast, -1, 3},     // seq_* skills are SQ
		{d2monster.ModeSkill1, 1, 0},    // unused explicit slot falls back to the mode
		{d2monster.ModeAttack1, 2, 2},   // Cast of an A1-mode skill in slot 2 keeps the slot
	}

	for _, tt := range tests {
		if got := skillSlotFor(&p, tt.mode, tt.slot); got != tt.want {
			t.Errorf("mode %v slot %d: %d want %d", tt.mode, tt.slot, got, tt.want)
		}
	}
}

func TestColumnAttack(t *testing.T) {
	v := &d2mapentity.MonsterVitals{
		A1: d2mapentity.MonsterAttack{Min: 1, Max: 2}, A2: d2mapentity.MonsterAttack{Min: 3, Max: 4},
		S1: d2mapentity.MonsterAttack{Min: 5, Max: 6},
	}

	want := map[d2monster.Mode]int{
		d2monster.ModeAttack1: 1, d2monster.ModeAttack2: 3, d2monster.ModeSkill1: 5, d2monster.ModeSpecialCast: 5,
		d2monster.ModeSkill2: 1, d2monster.ModeSkill3: 1, d2monster.ModeSkill4: 1, d2monster.ModeCast: 1,
	}

	for m, w := range want {
		if got := columnAttack(v, m).Min; got != w {
			t.Errorf("mode %v: column min %d want %d", m, got, w)
		}
	}
}

func TestApplyElemResist(t *testing.T) {
	tot := &d2statlist.Totals{MagicResist: 20}
	tot.ResistShown = [4]int{75, 50, -25, 0}

	tests := []struct {
		e    int
		elem string
		t    *d2statlist.Totals
		want int
	}{
		{100, "fire", tot, 25}, {100, "cold", tot, 50}, {100, "ltng", tot, 125}, {100, "pois", tot, 100},
		{100, "mag", tot, 80}, {100, "fire", nil, 100}, {7, "fire", tot, 1},
	}

	for _, tt := range tests {
		if got := applyElemResist(tt.e, tt.elem, tt.t); got != tt.want {
			t.Errorf("%d %s: %d want %d", tt.e, tt.elem, got, tt.want)
		}
	}
}
