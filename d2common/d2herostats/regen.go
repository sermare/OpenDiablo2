package d2herostats

// Natural life and mana regeneration of a hero, from the exe (observations
// only). The server runs one "vitals" event per unit per game frame (event
// type 3, handler 0x0057e800, which re-schedules itself for frame+1): when
// the unit is not dead it runs, in this order, the life routine 0x0057e600,
// the stamina routine 0x0057e4f0 (see stamina.go on the stamina branch) and
// the mana routine 0x0057e6e0. All three work in 1/256 units ("raw") of the
// life/mana/stamina stats (stat ids 6, 8, 10).
//
// Frame rate: the server tick is 1000/fps ms with fps read from 0x0072ee74,
// whose initial value is 25 (VERIFIED in the image; 0x006cc266 computes the
// tick), so the routines run 25 times per second.

// RegenFrameHz is the number of server frames per second the routines run at.
const RegenFrameHz = 25

// manaRegenFallbackFrames is the divisor the exe uses when the class has no
// ManaRegen (0x1d4c = 7500 frames = 300 s). No vanilla class hits it.
const manaRegenFallbackFrames = 0x1d4c

// lifeFloorRaw is the minimum life (1 point) the life routine leaves in place
// when it applies a regeneration (0x0057e64b: "if life < 0x100 then 0x100").
const lifeFloorRaw = 0x100

// ManaRegenPerFrame is the raw mana one frame of natural regeneration adds
// (VERIFIED 0x0057e6e0, read from the disassembly; maxRaw is GetMaxMana, the
// raw maximum):
//
//	base = 0
//	if the unit lacks state 85 (STATE_NOMANAREGEN):
//	    frames = charstats ManaRegen * 25          (7500 when ManaRegen is 0)
//	    base   = max(1, maxRaw / frames)           (integer division)
//	    base   = base * (100 + stat27) / 100       (MulDiv, truncating; stat 27
//	                                                is manarecoverybonus, the
//	                                                "regenerate mana %" stat)
//	total = base + stat26                          (manarecovery, a flat raw
//	                                                per frame amount that the
//	                                                mana potion states carry;
//	                                                added even under state 85)
//
// So with ManaRegen 120 a hero regains its whole maximum in 120 s (3000
// frames); "+100% regenerate mana" halves that. The caller clamps the result
// to [-mana, max-mana] with ClampVitalDelta.
func ManaRegenPerFrame(maxRaw, regenSeconds, bonusPct, flatRaw int, noRegenState bool) int {
	total := 0

	if !noRegenState {
		frames := regenSeconds * RegenFrameHz
		if frames == 0 {
			frames = manaRegenFallbackFrames
		}

		base := maxRaw / frames
		if base < 1 {
			base = 1
		}

		total = base * (100 + bonusPct) / 100
	}

	return total + flatRaw
}

// ClampVitalDelta limits a mana change as 0x0057e7b1 does: the result never
// exceeds the maximum and never goes below zero.
func ClampVitalDelta(curRaw, maxRaw, delta int) int {
	if delta > maxRaw-curRaw {
		delta = maxRaw - curRaw
	}

	if delta < -curRaw {
		delta = -curRaw
	}

	return delta
}

// LifeRegenStep applies the life routine once (VERIFIED 0x0057e600) and
// returns the new raw life. regenRaw is stat 74 (hpregen, "replenish life"),
// the raw life added per frame; when it is 0 the routine does nothing. The
// sum is capped at the maximum and a regeneration never leaves the hero under
// 1 point of life. Natural life regeneration does not exist: a hero without
// the stat (items, potions, skills) gets nothing, and 10 points of replenish
// life are 10*25/256 = 0.98 life per second.
func LifeRegenStep(curRaw, maxRaw, regenRaw int) int {
	if regenRaw == 0 {
		return curRaw
	}

	life := curRaw + regenRaw
	if life > maxRaw {
		life = maxRaw
	}

	if life < lifeFloorRaw {
		life = lifeFloorRaw
	}

	return life
}

// RegenInput is what the regeneration needs from the hero, in whole points
// for the maxima and the stat values as the exe holds them.
type RegenInput struct {
	MaxLife, MaxMana int // whole points (the exe's maxima are raw; the fraction is lost)
	ManaRegenSeconds int // charstats ManaRegen of the class
	ManaBonusPct     int // stat 27 (items, shrine)
	ManaFlatRaw      int // stat 26 (raw per frame)
	LifeRegenRaw     int // stat 74 (raw per frame)
	NoManaRegen      bool
}

// Regen advances the hero's life and mana in real time. It keeps the sub
// frame time and the fractions (below one point) between calls.
type Regen struct {
	frames     float64
	lifeFrac   int // raw life below one point, carried between calls
	manaFrac   int
	TotalTicks int // frames run so far (for tests and logs)
}

// maxCatchUpFrames bounds the frames one Advance runs after a long stall.
const maxCatchUpFrames = 250

// Advance runs the frames that the elapsed seconds contain on life and mana
// (whole points, updated in place) and returns how many it ran. The hero must
// be alive: the caller skips a dead hero as the exe does.
func (r *Regen) Advance(elapsed float64, in RegenInput, life, mana *int) int {
	if elapsed <= 0 {
		return 0
	}

	r.frames += elapsed * RegenFrameHz

	n := int(r.frames)
	if n > maxCatchUpFrames {
		n = maxCatchUpFrames
		r.frames = float64(n)
	}

	r.frames -= float64(n)
	r.TotalTicks += n

	lifeRaw, manaRaw := *life<<8+r.lifeFrac, *mana<<8+r.manaFrac
	maxLifeRaw, maxManaRaw := in.MaxLife<<8, in.MaxMana<<8

	for i := 0; i < n; i++ {
		if in.MaxLife > 0 {
			lifeRaw = LifeRegenStep(lifeRaw, maxLifeRaw, in.LifeRegenRaw)
		}

		if in.MaxMana > 0 {
			d := ManaRegenPerFrame(maxManaRaw, in.ManaRegenSeconds, in.ManaBonusPct, in.ManaFlatRaw, in.NoManaRegen)
			manaRaw += ClampVitalDelta(manaRaw, maxManaRaw, d)
		}
	}

	*life, r.lifeFrac = lifeRaw>>8, lifeRaw&0xff
	*mana, r.manaFrac = manaRaw>>8, manaRaw&0xff

	// a hero at its maximum carries no fraction (it would show 1 point over)
	if in.MaxLife > 0 && *life >= in.MaxLife {
		*life, r.lifeFrac = in.MaxLife, 0
	}

	if in.MaxMana > 0 && *mana >= in.MaxMana {
		*mana, r.manaFrac = in.MaxMana, 0
	}

	return n
}
