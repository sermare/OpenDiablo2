// Package d2animspeed is a pure model of how the original executable turns
// the speed stats (IAS, FCR, FHR, FBR, FRW) into the animation rate of a unit.
//
// Source: the exe function that recomputes a unit's velocity and animation
// rate when a speed stat changes (see ~/git/d2-re-notes/anim-speed-oracle.md).
// Everything marked VERIFIED was read from that function and its stat table;
// everything marked UNVERIFIED is a community rule or an inference.
//
// The animation rate is the 8.8 fixed point value the engine adds to a frame
// accumulator once per 25 Hz tick (256 = one frame per tick). The base rate is
// the AnimData speed of the animation (d2animdata Speed()).
package d2animspeed

import "errors"

// Stat-table constants (VERIFIED: the 12 byte rows at 0x6ea3d4 are
// {diminish flag, K, stat id}; the flag is 1 for all five rows).
const (
	// KStandard is the diminishing-returns constant of IAS, FHR, FCR and FBR.
	KStandard = 120
	// KRunWalk is the constant of faster run/walk.
	KRunWalk = 150
)

// Rate clamps (VERIFIED).
const (
	// MinActionPct and MaxActionPct clamp the attack percent (15..175) and the
	// generic "other" percent (stat 0x45).
	MinActionPct = 15
	MaxActionPct = 175
	// MinWalkPct is the lowest run/walk percent (25).
	MinWalkPct = 25
	// MaxRate is the largest rate the exe stores (0x7fff).
	MaxRate = 0x7fff
	// HitBasePct and BlockBasePct are the percent of the AnimData speed a
	// hit-recovery or block animation plays at with no modifiers (VERIFIED: 50).
	HitBasePct   = 50
	BlockBasePct = 50
	// BlockBaseStatePct replaces BlockBasePct while the unit has state 0x65
	// (VERIFIED in code; the meaning of state 0x65 is UNVERIFIED).
	BlockBaseStatePct = 100
	// CastBasePct is the base percent of a cast animation (VERIFIED: 100).
	CastBasePct = 100
	// WalkBasePct is the value stat 0x43 holds with no modifiers
	// (UNVERIFIED: assumed 100, the exe adds the FRW diminished bonus to it).
	WalkBasePct = 100
	// AttackBasePct is stat 0x44 with a weapon of WSM 0 and no IAS (UNVERIFIED
	// composition; the exe clamps stat0x44 + diminished item IAS to 15..175).
	AttackBasePct = 100
	// KickAdjust is added to the attack percent when the unit's raw mode field is
	// 0x12 (VERIFIED value). NOTE (oracle): player kick (mode 12) does NOT get
	// it; the oracle shows modes 7, 8, 11 and 12 all play at plain attack
	// rates, and mode 0x12 (the sequence mode) is not an attack, so the term
	// only fires if a form change makes the effective mode an attack while the
	// raw mode is 0x12 (UNVERIFIED which case that is).
	KickAdjust = -30
)

// Diminish is the diminishing-returns step of the exe's stat table lookup
// (VERIFIED): k*p/(k+p) with C truncation, applied only when p != 0.
func Diminish(p, k int) int {
	if p == 0 || k+p == 0 {
		return p
	}

	return k * p / (k + p)
}

// scale computes base*pct/100 the way the exe does (unsigned, truncating),
// treating non-positive results as 0 and capping at MaxRate.
func scale(base, pct int) int {
	if base <= 0 || pct <= 0 {
		return 0
	}

	r := base * pct / 100
	if r > MaxRate-1 {
		return MaxRate
	}

	return r
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}

	if v > hi {
		return hi
	}

	return v
}

// AttackRate is the rate of an attack animation (VERIFIED structure). pct is
// stat0x44 (UNVERIFIED: 100 minus the weapon WSM plus skill speed) plus the
// diminished item IAS; kick applies the assassin -30. dualAvgAdjust is the
// dual wield term (average of both weapons' stat 0x44 minus the first one's).
func AttackRate(base, stat44, itemIAS, dualAvgAdjust int, kick bool) int {
	pct := stat44 + Diminish(itemIAS, KStandard) + dualAvgAdjust
	if kick {
		pct += KickAdjust
	}

	return scale(base, clamp(pct, MinActionPct, MaxActionPct))
}

// CastRate is the rate of a spell cast animation (VERIFIED): the percent is
// 100 plus diminished FCR, capped at 175 (there is no lower clamp).
func CastRate(base, fcr int) int {
	pct := CastBasePct + Diminish(fcr, KStandard)
	if pct > MaxActionPct {
		pct = MaxActionPct
	}

	return scale(base, pct)
}

// HitRate is the rate of the hit recovery animation (VERIFIED): the percent
// is 50 plus diminished FHR, with no clamp. Note the 50: with no FHR the
// animation plays at half the AnimData speed.
func HitRate(base, fhr int) int {
	return scale(base, HitBasePct+Diminish(fhr, KStandard))
}

// BlockRate is the rate of the block recovery animation (VERIFIED): 50 (or
// 100 with state 0x65) plus diminished FBR; at least 1 when base > 0.
func BlockRate(base, fbr int, state65 bool) int {
	b := BlockBasePct
	if state65 {
		b = BlockBaseStatePct
	}

	r := scale(base, b+Diminish(fbr, KStandard))
	if r == 0 && base > 0 {
		r = 1
	}

	return r
}

// WalkRate is the animation rate of walking or running (VERIFIED): the
// percent is stat 0x43 plus diminished FRW (K=150), at least 25.
func WalkRate(base, stat43Bonus, frw int) int {
	pct := WalkBasePct + stat43Bonus + Diminish(frw, KRunWalk)
	if pct < MinWalkPct {
		pct = MinWalkPct
	}

	return scale(base, pct)
}

// WalkVelocity is the path velocity of walking or running: baseVelocity*pct/100
// with the same percent as WalkRate (VERIFIED structure).
func WalkVelocity(baseVelocity, stat43Bonus, frw int) int {
	pct := WalkBasePct + stat43Bonus + Diminish(frw, KRunWalk)
	if pct < MinWalkPct {
		pct = MinWalkPct
	}

	return baseVelocity * pct / 100
}

// OtherRate is the generic rate (stat 0x45 clamped to 15..175, VERIFIED).
func OtherRate(base, stat45 int) int {
	return scale(base, clamp(stat45, MinActionPct, MaxActionPct))
}

// ErrNoRate is returned when an action cannot complete (zero rate).
var ErrNoRate = errors.New("d2animspeed: zero animation rate")

// Ticks returns the number of 25 Hz ticks an animation of `frames` frames
// takes at the 8.8 rate: ceil(256*frames/rate). VERIFIED to equal what the
// engine's accumulator stepper produces (d2asset tickStepper).
func Ticks(frames, rate int) (int, error) {
	if rate <= 0 {
		return 0, ErrNoRate
	}

	return (256*frames + rate - 1) / rate, nil
}

// ActionFrames is Ticks minus one, the number the community tables list for
// attack, hit recovery, cast and block (UNVERIFIED in the exe: the breakpoint
// thresholds do not depend on the -1, only the absolute count does).
func ActionFrames(frames, rate int) (int, error) {
	t, err := Ticks(frames, rate)
	if err != nil {
		return 0, err
	}

	return t - 1, nil
}

// Breakpoints returns the smallest stat value at which fn's result first
// changes, starting with 0, up to `entries` entries (or maxStat).
func Breakpoints(fn func(stat int) int, entries, maxStat int) []int {
	out := make([]int, 0, entries)
	cur := -1

	for p := 0; p <= maxStat && len(out) < entries; p++ {
		if f := fn(p); f != cur {
			out = append(out, p)
			cur = f
		}
	}

	return out
}

// Mode is the AnimData record of a class animation: frames per direction and
// base speed (VERIFIED from the expansion animdata.d2, hand-to-hand token).
type Mode struct{ Frames, Speed int }

// ClassModes lists, per class, the AnimData of walk, run, hit, block, first
// attack and spell cast with bare hands (token+mode+"HTH"). Weapon classes
// change the attack and cast numbers (e.g. AMA11HS is 16 frames).
type ClassModes struct {
	Walk, Run, Hit, Block, Attack1, Cast Mode
}

// Classes holds the AnimData of the seven classes (VERIFIED against d2exp
// animdata.d2; the engine reads the same numbers at runtime).
var Classes = map[string]ClassModes{
	"Amazon":      {Mode{8, 256}, Mode{8, 256}, Mode{6, 256}, Mode{3, 256}, Mode{13, 256}, Mode{20, 256}},
	"Sorceress":   {Mode{8, 256}, Mode{8, 256}, Mode{8, 256}, Mode{5, 256}, Mode{16, 256}, Mode{14, 256}},
	"Necromancer": {Mode{8, 256}, Mode{8, 256}, Mode{7, 256}, Mode{6, 256}, Mode{15, 256}, Mode{16, 256}},
	"Paladin":     {Mode{10, 288}, Mode{8, 256}, Mode{5, 256}, Mode{3, 256}, Mode{14, 256}, Mode{16, 256}},
	"Barbarian":   {Mode{8, 168}, Mode{8, 216}, Mode{5, 256}, Mode{4, 256}, Mode{12, 256}, Mode{14, 256}},
	"Druid":       {Mode{8, 136}, Mode{8, 168}, Mode{7, 256}, Mode{6, 256}, Mode{16, 256}, Mode{15, 208}},
	"Assassin":    {Mode{8, 256}, Mode{8, 256}, Mode{5, 256}, Mode{3, 256}, Mode{11, 256}, Mode{17, 256}},
}
