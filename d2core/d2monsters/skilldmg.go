package d2monsters

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2skill"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Monster skill damage. A monster's attack mode is chosen by its AI; what the
// mode DOES comes from the skill in the monstats slot (Skill1..8 with Sk#mode
// and Sk#lvl), see d2common/d2monster/attackdmg.go for the verified rules:
// the unit damage of a mode comes from the A1/A2/S1 columns (ColumnForMode),
// and a skills.txt row decides whether that damage is dealt (melee kinds,
// SrcDam percent), replaced by the row's own columns (area kinds), or handed
// to a missile (own missile columns, or the row of the missile's Skill).
//
// UNVERIFIED simplifications: the DmgSymPerCalc synergy programs are ignored
// (a monster has no skill points, they evaluate to 0 in practice); area
// effects hit the AI target without a radius test; summons, curses and
// teleports deal no damage here (EffectNone).

// monSkill is the resolved skill of a monster attack request.
type monSkill struct {
	slot    int
	name    string
	level   int
	rec     *d2records.SkillRecord
	kind    d2monster.EffectKind
	missile string // srvmissile of the row ("" = none)
}

// skillSlotFor picks the monstats slot of a request: the explicit slot of a
// Cast, else the first used slot whose Sk#mode equals the mode (UNVERIFIED
// for AI code that names a mode without a slot). A1 and A2 without a slot are
// the plain attack and have no skill.
func skillSlotFor(p *d2monster.Profile, mode d2monster.Mode, slot int) int {
	if slot >= 0 && slot < d2monster.NumSkills && p.Skills[slot].Used() {
		return slot
	}

	if mode == d2monster.ModeAttack1 || mode == d2monster.ModeAttack2 {
		return -1
	}

	for i := range p.Skills {
		if p.Skills[i].Used() && p.Skills[i].Mode == mode {
			return i
		}
	}

	return -1
}

// resolveSkill looks the slot's skills.txt row up; nil when the slot is
// absent or the row unknown.
func (d *Director) resolveSkill(p *d2monster.Profile, slot int) *monSkill {
	if slot < 0 || d.asset == nil || d.asset.Records == nil {
		return nil
	}

	s := p.Skills[slot]

	rec := d.asset.Records.GetSkillByName(s.Name)
	if rec == nil {
		return nil
	}

	lvl := s.Level
	if lvl < 1 {
		lvl = 1
	}

	missile := rec.Srvmissile

	return &monSkill{
		slot: slot, name: s.Name, level: lvl, rec: rec, missile: missile,
		kind: d2monster.ClassifyEffect(rec.Srvdofunc, missile != ""),
	}
}

func columnAttack(v *d2mapentity.MonsterVitals, mode d2monster.Mode) d2mapentity.MonsterAttack {
	switch d2monster.ColumnForMode(mode) {
	case d2monster.ColumnA2:
		return v.A2
	case d2monster.ColumnS1:
		return v.S1
	}

	return v.A1
}

// specFromSkill builds the damage columns of a skills.txt row.
func specFromSkill(r *d2records.SkillRecord) d2skill.DamageSpec {
	return d2skill.DamageSpec{
		HitShift: r.HitShift, SrcDam: r.SrcDam,
		MinDam: r.MinDam, MaxDam: r.MaxDam,
		MinLevDam: [5]int{r.MinLevDam1, r.MinLevDam2, r.MinLevDam3, r.MinLevDam4, r.MinLevDam5},
		MaxLevDam: [5]int{r.MaxLevDam1, r.MaxLevDam2, r.MaxLevDam3, r.MaxLevDam4, r.MaxLevDam5},
		EType:     r.EType, EMin: r.EMin, EMax: r.EMax,
		EMinLev: [5]int{r.EMinLev1, r.EMinLev2, r.EMinLev3, r.EMinLev4, r.EMinLev5},
		EMaxLev: [5]int{r.EMaxLev1, r.EMaxLev2, r.EMaxLev3, r.EMaxLev4, r.EMaxLev5},
		ELen:    r.ELen,
	}
}

// specFromMissile builds the damage columns of a missiles.txt row that has
// no Skill link (its own MinDamage/MaxDamage/EMin/Emax columns).
func specFromMissile(m *d2records.MissileRecord) d2skill.DamageSpec {
	return d2skill.DamageSpec{
		HitShift: m.HitShift, SrcDam: m.SourceDamage,
		MinDam: m.Damage.MinDamage, MaxDam: m.Damage.MaxDamage,
		MinLevDam: m.Damage.MinLevelDamage, MaxLevDam: m.Damage.MaxLevelDamage,
		EType: m.ElementalDamage.ElementType,
		EMin:  m.ElementalDamage.Damage.MinDamage, EMax: m.ElementalDamage.Damage.MaxDamage,
		EMinLev: m.ElementalDamage.Damage.MinLevelDamage, EMaxLev: m.ElementalDamage.Damage.MaxLevelDamage,
		ELen: m.ElementalDamage.Duration,
	}
}

// ownDamage turns a damage spec into an attack: the unit damage of the mode
// scaled by SrcDam/128, plus the row's physical columns, plus its element.
// Integer damage is the 8.8 value shifted back.
func ownDamage(spec *d2skill.DamageSpec, lvl int, base d2mapentity.MonsterAttack) d2mapentity.MonsterAttack {
	var env *d2skill.Env // nil programs never evaluate

	atk := d2mapentity.MonsterAttack{ToHit: base.ToHit}

	if spec.SrcDam > 0 {
		atk.Min = base.Min * spec.SrcDam / 128
		atk.Max = base.Max * spec.SrcDam / 128
	}

	atk.Min += int(spec.PhysMin(env, lvl, 0, false)) >> 8
	atk.Max += int(spec.PhysMax(env, lvl, 0, false)) >> 8

	if et := normElem(spec.EType); et != "" {
		atk.ElemType = et
		atk.ElemMin = int(spec.ElemMin(env, lvl)) >> 8
		atk.ElemMax = int(spec.ElemMax(env, lvl)) >> 8

		if et == "pois" {
			// poison columns are a per-frame rate in 8.8; the total is rate *
			// ELen frames (UNVERIFIED: dealt at once here, not over time)
			n := spec.ElemLen(env, lvl)
			atk.ElemMin = int(spec.ElemMin(env, lvl)) * n >> 8
			atk.ElemMax = int(spec.ElemMax(env, lvl)) * n >> 8
		}

		if atk.ElemMax < atk.ElemMin {
			atk.ElemMax = atk.ElemMin
		}
	}

	if atk.Max < atk.Min {
		atk.Max = atk.Min
	}

	return atk
}

// normElem maps a skills.txt/missiles.txt EType token to the element names of
// MonsterAttack; stun, life steal and the like are not damage here.
func normElem(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "fire":
		return "fire"
	case "cold", "frze":
		return "cold"
	case "ltng":
		return "ltng"
	case "mag":
		return "mag"
	case "pois":
		return "pois"
	}

	return ""
}

// skillAttack is the damage profile of a monster skill cast in a mode, and
// false when the skill deals no direct damage.
func (d *Director) skillAttack(v *d2mapentity.MonsterVitals, sk *monSkill, mode d2monster.Mode) (d2mapentity.MonsterAttack, bool) {
	var (
		mr *d2records.MissileRecord
		lr *d2records.SkillRecord
	)

	if sk.kind == d2monster.EffectMissile {
		if mr = d.asset.Records.GetMissileByName(sk.missile); mr != nil && mr.SkillName != "" {
			lr = d.asset.Records.GetSkillByName(mr.SkillName)
		}
	}

	return skillDamage(columnAttack(v, mode), sk, mr, lr)
}

// skillDamage is the pure core of skillAttack: base is the unit damage of the
// mode, mr the srvmissile row (missile kinds) and lr the row of that missile's
// Skill column (nil when it has none).
func skillDamage(base d2mapentity.MonsterAttack, sk *monSkill, mr *d2records.MissileRecord,
	lr *d2records.SkillRecord) (d2mapentity.MonsterAttack, bool) {
	var spec d2skill.DamageSpec

	switch sk.kind {
	case d2monster.EffectMelee, d2monster.EffectArea:
		spec = specFromSkill(sk.rec)
	case d2monster.EffectMissile:
		spec = specFromSkill(sk.rec)

		switch {
		case lr != nil:
			// a Skill-linked missile reads the damage of that skills.txt row
			spec = specFromSkill(lr)
		case mr != nil && mr.SkillName == "":
			spec = specFromMissile(mr)
		}
	default:
		return d2mapentity.MonsterAttack{}, false
	}

	atk := ownDamage(&spec, sk.level, base)

	return atk, atk.Max > 0 || atk.ElemMax > 0
}

// attackOf is the damage profile of the attack the unit is making: its skill
// when the request bound one, else the plain A1/A2 attack.
func (d *Director) attackOf(u *unit, mode d2monster.Mode) (d2mapentity.MonsterAttack, bool) {
	if u.skill != nil {
		return d.skillAttack(&u.m.Vitals, u.skill, mode)
	}

	return attackFor(&u.m.Vitals, mode)
}

// attackFlies says whether the attack launches a projectile: the skill names
// a srvmissile, or the monstats mode does.
func (d *Director) attackFlies(u *unit, mode d2monster.Mode) bool {
	if u.skill != nil && u.skill.missile != "" {
		return true
	}

	return attackIsRanged(u.m.Stat, mode)
}

// shotMissile is the missile of a shot: the skill's srvmissile, else the
// monstats Miss column of the mode.
func shotMissile(u *unit, mode d2monster.Mode) string {
	if u.skill != nil && u.skill.missile != "" {
		return u.skill.missile
	}

	return missileFor(u.m.Stat, mode)
}

// elementalHit rolls the element part of an attack against the hero's
// resistance (ResistShown: fire, cold, lightning, poison; magic has its own
// stat). Returns the damage to add; a roll is consumed only when there is an
// element.
func (d *Director) elementalHit(u *unit, p *d2mapentity.Player, atk d2mapentity.MonsterAttack) int {
	if atk.ElemType == "" || atk.ElemMax <= 0 {
		return 0
	}

	// El#Pct: a chance below 100 is one percent roll on the unit's seed (0x5a2960), made before the damage
	if atk.ElemPct > 0 && int(u.b.Seed.Roll(100)) >= atk.ElemPct {
		return 0
	}

	e := atk.ElemMin + int(u.b.Seed.Roll(int32(atk.ElemMax-atk.ElemMin+1)))

	return applyElemResist(e, atk.ElemType, p.Stats.Totals)
}

// applyElemResist reduces an elemental roll by the hero's shown resistance of
// that element (negative resistance adds damage); nil totals mean 0.
func applyElemResist(e int, elem string, t *d2statlist.Totals) int {
	pct := 0

	if t != nil {
		switch elem {
		case "fire":
			pct = t.ResistShown[d2statlist.ResFire]
		case "cold":
			pct = t.ResistShown[d2statlist.ResCold]
		case "ltng":
			pct = t.ResistShown[d2statlist.ResLight]
		case "pois":
			pct = t.ResistShown[d2statlist.ResPoison]
		case "mag":
			pct = t.MagicResist
		}
	}

	return e * (100 - pct) / 100
}
