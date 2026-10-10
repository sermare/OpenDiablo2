package d2herostats

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"

// Pool is a current/maximum pair in the exe's 1/256 fixed point (stat 6/7
// life, 8/9 mana, 10/11 stamina).
type Pool struct{ Cur, Max int }

// Resources are a hero's three pools.
type Resources struct{ Life, Mana, Stamina Pool }

// ApplyVitality spends n vitality points (n < 0 refunds them).
//
// VERIFIED 0x0056ea50 (stat allocation, also used by the stat reset): max life
// += charstats LifePerVitality * n * 0x40 and max stamina += StaminaPerVitality
// * n * 0x40; the current value gets the same grant only when n > 0; a current
// value above the new maximum is clamped to it.
func ApplyVitality(c d2statlist.Class, r Resources, n int) Resources {
	r.Life = grow(r.Life, c.LifePerVit*n*0x40, n)
	r.Stamina = grow(r.Stamina, c.StaminaPerVit*n*0x40, n)

	return r
}

// ApplyEnergy spends n energy points: VERIFIED 0x0056e970, max mana +=
// charstats ManaPerEnergy * n * 0x40, current mana likewise only when n > 0,
// clamped to the maximum.
func ApplyEnergy(c d2statlist.Class, r Resources, n int) Resources {
	r.Mana = grow(r.Mana, c.ManaPerEne*n*0x40, n)

	return r
}

func grow(p Pool, delta, n int) Pool {
	p.Max += delta
	if n > 0 {
		p.Cur += delta
	}

	if p.Cur > p.Max {
		p.Cur = p.Max
	}

	return p
}
