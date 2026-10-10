package d2statlist

import (
	"fmt"
	"sort"
	"strings"
)

// Stat ids used by the derivations (ItemStatCost.txt "ID" column, verified
// against the 1.14b table).
const (
	StatStrength     = 0
	StatEnergy       = 1
	StatDexterity    = 2
	StatVitality     = 3
	StatMaxHP        = 7
	StatMaxMana      = 9
	StatMaxStamina   = 11
	StatArmorPct     = 16 // item_armor_percent, enhanced defense (item specific)
	StatMaxDmgPct    = 17 // item_maxdamage_percent (enhanced damage)
	StatMinDmgPct    = 18
	StatToHit        = 19
	StatToBlock      = 20
	StatMinDamage    = 21
	StatMaxDamage    = 22
	StatSecMinDamage = 23
	StatSecMaxDamage = 24
	StatDamagePct    = 25
	StatArmorClass   = 31
	StatIgnoreDef    = 0x73 // item_ignoretargetac (attack ignores target defense)
	StatTargetACPct  = 0x74 // item_fractionaltargetac (percent of target defense removed)
	StatDemonAR      = 0x7b // item_demon_tohit
	StatUndeadAR     = 0x7c // item_undead_tohit
	StatArmorMissile = 32
	StatArmorHTH     = 33
	StatNormalReduce = 34 // normal_damage_reduction (flat physical damage reduction)
	StatMagicReduce  = 35
	StatDamageResist = 36 // physical resistance
	StatMagicResist  = 37
	StatMaxMagicRes  = 38
	StatFireResist   = 39
	StatMaxFireRes   = 40
	StatLightResist  = 41
	StatMaxLightRes  = 42
	StatColdResist   = 43
	StatMaxColdRes   = 44
	StatPoisonResist = 45
	StatMaxPoisonRes = 46
	StatFireMin      = 48
	StatFireMax      = 49
	StatLightMin     = 50
	StatLightMax     = 51
	StatColdMin      = 54
	StatColdMax      = 55
	StatPoisonMin    = 57
	StatPoisonMax    = 58
	StatLifeSteal    = 60
	StatManaSteal    = 62
	StatManaRecRaw   = 26 // manarecovery: raw mana per frame (potion states)
	StatManaRecovery = 27 // manarecoverybonus
	StatStamRecovery = 28
	StatHPRegen      = 74
	StatMaxHPPct     = 76
	StatMaxManaPct   = 77
	StatGoldFind     = 79
	StatMagicFind    = 80
	StatAddExp       = 85 // item_addexperience: +% experience from kills (ItemStatCost id 85)
	StatReduceReqPct = 91
	StatFasterAttack = 93
	StatFasterMove   = 96
	StatFasterHit    = 99
	StatFasterBlock  = 102
	StatFasterCast   = 105
	StatDamageToMana = 114
	StatToHitPct     = 119
	StatAllSkills    = 127
	StatClassSkills  = 83
	StatSingleSkill  = 107
	StatNonClass     = 97
	StatElemSkill    = 126
	StatSkillTab     = 188
	StatDeadlyStrike = 141
	StatCrushing     = 136
	StatOpenWounds   = 135
	StatCritical     = 337 // passive_critical_strike
	StatDodge        = 338
	StatAvoid        = 339
	StatEvade        = 340
	StatPierceCold   = 305
	StatPierceFire   = 306
	StatPierceLight  = 307
	StatPiercePoison = 308
)

// Prop is one stat contribution: a stat id, its parameter (skill id, class
// and tab, ...) and a value in the units an item shows.
type Prop struct {
	ID    int
	Param int
	Value int64
}

type key struct{ id, param int }

// List is a set of summed stats keyed by (stat id, parameter).
type List struct {
	v map[key]int64
}

// NewList returns an empty list.
func NewList() *List { return &List{v: map[key]int64{}} }

// Add adds delta to the stat (id, param).
func (l *List) Add(id, param int, delta int64) {
	if l.v == nil {
		l.v = map[key]int64{}
	}

	l.v[key{id, param}] += delta
}

// AddProps adds every property.
func (l *List) AddProps(props []Prop) {
	for _, p := range props {
		l.Add(p.ID, p.Param, p.Value)
	}
}

// Merge adds all stats of o.
func (l *List) Merge(o *List) {
	if o == nil {
		return
	}

	for k, v := range o.v {
		l.Add(k.id, k.param, v)
	}
}

// Get returns the stat with parameter 0.
func (l *List) Get(id int) int64 {
	if l == nil {
		return 0
	}

	return l.v[key{id, 0}]
}

// GetParam returns the stat for one parameter.
func (l *List) GetParam(id, param int) int64 {
	if l == nil {
		return 0
	}

	return l.v[key{id, param}]
}

// Sum returns the stat summed over all parameters.
func (l *List) Sum(id int) int64 {
	var s int64

	if l == nil {
		return 0
	}

	for k, v := range l.v {
		if k.id == id {
			s += v
		}
	}

	return s
}

// Len is the number of distinct (id, param) entries.
func (l *List) Len() int {
	if l == nil {
		return 0
	}

	return len(l.v)
}

// Props returns the entries sorted by id then parameter.
func (l *List) Props() []Prop {
	if l == nil {
		return nil
	}

	out := make([]Prop, 0, len(l.v))

	for k, v := range l.v {
		out = append(out, Prop{ID: k.id, Param: k.param, Value: v})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}

		return out[i].Param < out[j].Param
	})

	return out
}

// String lists the entries as "id[:param]=value" for logs.
func (l *List) String() string {
	var parts []string

	for _, p := range l.Props() {
		if p.Param != 0 {
			parts = append(parts, fmt.Sprintf("%d:%d=%d", p.ID, p.Param, p.Value))
		} else {
			parts = append(parts, fmt.Sprintf("%d=%d", p.ID, p.Value))
		}
	}

	return strings.Join(parts, " ")
}
