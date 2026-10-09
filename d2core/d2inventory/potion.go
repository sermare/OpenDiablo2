package d2inventory

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// Potion effects, from the misc.txt columns pSpell, len, stat1/calc1 and
// stat2/calc2 (the item tables, not the game's code). What the columns mean is
// read from the shipped rows, not verified in the binary:
//
//   - pSpell 3 (healing and mana potions): stat1 is hpregen / manarecovery and
//     calc1 the total amount (Greater Healing 320 life, Greater Mana 250 mana),
//     restored evenly over len frames (25 frames per second: 256 frames about
//     10 s). The original ticks the amount once per frame batch; an even
//     spread over the same time is the approximation used here.
//   - pSpell 5 (rejuvenation): stat1 hitpoints and stat2 mana with calc1 and
//     calc2 as a PERCENT of the maximum, applied at once (35 and 100).
//
// Other pSpell values (antidote, thawing, stamina, scrolls) have no effect here.

const potionFramesPerSecond = 25.0

const (
	spellOverTime = 3
	spellRejuv    = 5
)

// PotionEffect is what drinking a potion does.
type PotionEffect struct {
	// Instant amounts, in percent of the maximum.
	InstantHPPercent, InstantManaPercent int
	// Over time, in points, spread across Seconds.
	HP, Mana float64
	Seconds  float64
}

// IsEmpty reports whether the potion does nothing here.
func (e PotionEffect) IsEmpty() bool {
	return e.InstantHPPercent == 0 && e.InstantManaPercent == 0 && e.HP == 0 && e.Mana == 0
}

// PotionEffectOf reads the effect of a misc.txt potion.
func PotionEffectOf(rec *d2records.ItemCommonRecord) PotionEffect {
	var e PotionEffect

	if rec == nil || !rec.Useable {
		return e
	}

	switch rec.SpellType {
	case spellOverTime:
		e.Seconds = float64(rec.EffectLength) / potionFramesPerSecond

		for _, u := range rec.UsageStats {
			amount := calcNumber(string(u.Calc))

			switch strings.ToLower(u.Stat) {
			case "hpregen":
				e.HP += float64(amount)
			case "manarecovery":
				e.Mana += float64(amount)
			}
		}
	case spellRejuv:
		for _, u := range rec.UsageStats {
			amount := calcNumber(string(u.Calc))

			switch strings.ToLower(u.Stat) {
			case "hitpoints":
				e.InstantHPPercent += amount
			case "mana":
				e.InstantManaPercent += amount
			}
		}
	}

	return e
}

// calcNumber reads a calc column that holds a plain number (all potion rows).
func calcNumber(s string) int {
	n := 0

	for _, r := range strings.TrimSpace(s) {
		if r < '0' || r > '9' {
			return 0
		}

		n = n*10 + int(r-'0')
	}

	return n
}

// Regen is the over-time part of the potions the hero drank: points still to
// be restored and the speed.
type Regen struct {
	hp, mana         float64
	hpRate, manaRate float64
}

// Add starts the over-time part of an effect. A potion drunk while an earlier
// one is still working adds to the remaining points and raises the rate so
// both finish together (a simplification: the original runs each potion's
// timer separately).
func (r *Regen) Add(e PotionEffect) {
	if e.Seconds <= 0 || (e.HP == 0 && e.Mana == 0) {
		return
	}

	r.hp += e.HP
	r.mana += e.Mana
	r.hpRate += e.HP / e.Seconds
	r.manaRate += e.Mana / e.Seconds
}

// Active reports whether anything is left to restore.
func (r *Regen) Active() bool { return r.hp > 0 || r.mana > 0 }

// Tick advances dt seconds and returns the points restored in that time.
func (r *Regen) Tick(dt float64) (hp, mana float64) {
	hp, r.hp = takeRegen(r.hp, r.hpRate*dt)
	mana, r.mana = takeRegen(r.mana, r.manaRate*dt)

	if r.hp <= 0 {
		r.hpRate = 0
	}

	if r.mana <= 0 {
		r.manaRate = 0
	}

	return hp, mana
}

func takeRegen(left, step float64) (taken, remaining float64) {
	if step > left {
		step = left
	}

	return step, left - step
}

// ApplyInstant applies the instant part of an effect to a pool and returns the
// new value (clamped to max).
func ApplyInstant(current, max, percent int) int {
	v := current + max*percent/100
	if v > max {
		return max
	}

	return v
}
