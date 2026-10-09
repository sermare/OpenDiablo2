package d2object

import (
	"fmt"
	"sort"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"
)

// Stat ids of the shrine states that d2statlist has no constant for.
// VERIFIED as the stat of each STATE_SHRINE_* in the effect table of Game.exe
// (0x6e2b48): 171 skill_armor_percent, 162 skill_staminapercent, 85 experience
// gain, 27 manarecoverybonus, 39/41/43/45 the resists.
const (
	StatSkillArmorPct   = 171
	StatSkillStaminaPct = 162
	StatExperience      = 85
	// StatPoisonLengthReduce is item_poisonlength_resist (UNVERIFIED id use:
	// the poison shrine description says "poison duration = 0").
	StatPoisonLengthReduce = 110
)

// State ids (states.txt rows) of the shrine buffs. VERIFIED from the table
// of Game.exe (third word of each effect entry).
const (
	StateShrineArmor     = 128
	StateShrineCombat    = 129
	StateShrineResistLtn = 130
	StateShrineResistFir = 131
	StateShrineResistCld = 132
	StateShrineResistPoi = 133
	StateShrineSkill     = 134
	StateShrineManaRegen = 135
	StateShrineStamina   = 136
	StateShrineExp       = 137
)

// Buff is a timed stat list on the hero.
type Buff struct {
	Source string // "shrine:Armor Boost"
	State  int    // states.txt id; a new buff replaces the old one of the same state
	Stats  *d2statlist.List
	// Expires is the game time in seconds at which the buff ends.
	Expires float64
}

// ShrineState returns the state id and the stat list of a booster shrine, or
// ok=false when the shrine is not a timed booster.
func ShrineState(s Shrine) (state int, stats *d2statlist.List, ok bool) {
	l := d2statlist.NewList()

	switch s.Code {
	case ShrineArmor:
		state = StateShrineArmor

		l.Add(StatSkillArmorPct, 0, int64(s.Arg0))
	case ShrineCombat:
		// special function 0x5818b0 in the binary (not decoded): +Arg0% to hit
		// and +Arg1% damage as the shrines.txt description says. UNVERIFIED stats.
		state = StateShrineCombat

		l.Add(d2statlist.StatToHitPct, 0, int64(s.Arg0))
		l.Add(d2statlist.StatDamagePct, 0, int64(s.Arg1))
	case ShrineResistFire:
		state = StateShrineResistFir

		l.Add(d2statlist.StatFireResist, 0, int64(s.Arg0))
	case ShrineResistCold:
		state = StateShrineResistCld

		l.Add(d2statlist.StatColdResist, 0, int64(s.Arg0))
	case ShrineResistLightning:
		state = StateShrineResistLtn

		l.Add(d2statlist.StatLightResist, 0, int64(s.Arg0))
	case ShrineResistPoison:
		state = StateShrineResistPoi

		l.Add(d2statlist.StatPoisonResist, 0, int64(s.Arg0))
		l.Add(StatPoisonLengthReduce, 0, 100)
	case ShrineSkill:
		state = StateShrineSkill

		l.Add(d2statlist.StatAllSkills, 0, int64(s.Arg0))
	case ShrineManaRecharge:
		state = StateShrineManaRegen

		l.Add(d2statlist.StatManaRecovery, 0, int64(s.Arg0))
	case ShrineStamina:
		state = StateShrineStamina

		l.Add(StatSkillStaminaPct, 0, int64(s.Arg0))
	case ShrineExperience:
		state = StateShrineExp

		// shrines.txt Arg0 is 0 here; the description says +50%. UNVERIFIED.
		l.Add(StatExperience, 0, 50)
	default:
		return 0, nil, false
	}

	return state, l, true
}

// NewShrineBuff makes the buff of a booster shrine used at game time now.
func NewShrineBuff(s Shrine, now float64) (Buff, bool) {
	st, l, ok := ShrineState(s)
	if !ok {
		return Buff{}, false
	}

	return Buff{Source: "shrine:" + s.Name, State: st, Stats: l, Expires: now + s.DurationSeconds()}, true
}

// Vitals are the current and maximum life and mana of the hero.
type Vitals struct {
	Life, MaxLife, Mana, MaxMana int
}

// ApplyInstant applies a recharge shrine (codes 1..5) to the vitals and
// returns the new vitals. ok is false for other shrine codes.
//
// Refill (VERIFIED by name only): both to maximum. The boost and exchange
// formulas follow the shrines.txt description ("doubles current health (even
// over maximum)", "takes 1/2 health, gives 5 times that number to mana") with
// Arg0/Arg1 as percentages; the caps (Arg1 percent of maximum) are UNVERIFIED.
func ApplyInstant(s Shrine, v Vitals) (Vitals, bool) {
	switch s.Code {
	case ShrineRefill:
		v.Life, v.Mana = v.MaxLife, v.MaxMana
	case ShrineHealth:
		v.Life = minInt(v.Life*s.Arg0/100, v.MaxLife*s.Arg1/100)
	case ShrineMana:
		v.Mana = minInt(v.Mana*s.Arg0/100, v.MaxMana*s.Arg1/100)
	case ShrineHealthExchange:
		taken := v.Life * s.Arg0 / 100
		v.Life -= taken
		v.Mana = minInt(v.Mana+taken*s.Arg1/100, v.MaxMana)
	case ShrineManaExchange:
		taken := v.Mana * s.Arg0 / 100
		v.Mana -= taken
		v.Life = minInt(v.Life+taken*s.Arg1/100, v.MaxLife)
	default:
		return v, false
	}

	return v, true
}

// WellRestore is what a well gives: full life and mana. The original's well
// function (OperateFn 22, 0x5837b0) is not decompiled; that it restores both
// and whether it also restores stamina is UNVERIFIED. objects.txt gives the
// wells Parm0=750: read here as the refill delay in frames (30 s), UNVERIFIED.
func WellRestore(v Vitals) Vitals {
	v.Life, v.Mana = v.MaxLife, v.MaxMana

	return v
}

// WellRefillSeconds is the time a used well stays empty, from objects.txt
// Parm0 (frames, default 750 when the column is 0).
func WellRefillSeconds(parm0 int) float64 {
	if parm0 <= 0 {
		parm0 = 750
	}

	return float64(parm0) / FramesPerSecond
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

// WorldEffect is a shrine effect that acts on the world instead of the hero's
// stats. The engine implements the ones it can; the others are logged as
// stubs.
type WorldEffect int

// World effects.
const (
	WorldNone WorldEffect = iota
	WorldPortal
	WorldGemUpgrade
	WorldStorm
	WorldWarping
	WorldExploding
	WorldPoison
	WorldEnirhs
)

var worldNames = [...]string{"none", "portal", "gem-upgrade", "storm", "warping", "exploding", "poison", "enirhs"}

func (w WorldEffect) String() string { return worldNames[w] }

// WorldFor returns the world effect of a magic shrine.
func WorldFor(s Shrine) WorldEffect {
	switch s.Code {
	case ShrinePortal:
		return WorldPortal
	case ShrineGem:
		return WorldGemUpgrade
	case ShrineStorm:
		return WorldStorm
	case ShrineWarping:
		return WorldWarping
	case ShrineExploding:
		return WorldExploding
	case ShrinePoison:
		return WorldPoison
	case ShrineEnirhs:
		return WorldEnirhs
	}

	return WorldNone
}

// Buffs is the list of active timed effects.
type Buffs struct {
	list []Buff
}

// Add starts a buff; a buff of the same state replaces (refreshes) the old one.
func (b *Buffs) Add(nb Buff) {
	for i := range b.list {
		if b.list[i].State == nb.State {
			b.list[i] = nb
			return
		}
	}

	b.list = append(b.list, nb)
}

// Expire removes the buffs that ended by time now and returns them.
func (b *Buffs) Expire(now float64) []Buff {
	var gone []Buff

	keep := b.list[:0]

	for _, x := range b.list {
		if x.Expires <= now {
			gone = append(gone, x)
		} else {
			keep = append(keep, x)
		}
	}

	b.list = keep

	return gone
}

// Active returns the running buffs sorted by state.
func (b *Buffs) Active() []Buff {
	out := append([]Buff(nil), b.list...)
	sort.Slice(out, func(i, j int) bool { return out[i].State < out[j].State })

	return out
}

// Total sums one stat over the active buffs.
func (b *Buffs) Total(stat int) int64 {
	var n int64

	for _, x := range b.list {
		n += x.Stats.Get(stat)
	}

	return n
}

// Describe is the log text of a buff: "Armor Boost state=128 stats=... ends=..".
func (x Buff) Describe(now float64) string {
	return fmt.Sprintf("%s state=%d stats=[%s] remaining=%.1fs", x.Source, x.State, x.Stats.String(), x.Expires-now)
}
