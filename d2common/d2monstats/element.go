package d2monstats

// Elemental attack columns El1..El3 of monstats.txt, scaled like the other groups of MONTBL_GetLevelScaledStats
// (0x6551e0, groups 0x40, 0x80, 0x100). VERIFIED against 0x5a2960 (MONSTER_SetupLevelScaledStats), which
// reads them whenever the requested unit mode equals the El#Mode byte:
//
//   - the slot is skipped when its chance (El#Pct of the difficulty) is 0; at 100 or more it always applies;
//     below that one percent roll under the chance decides;
//   - minimum and maximum are MulDiv(monlvl DM, percent, 100), raw when the class has noRatio; the length is
//     always raw;
//   - the type byte picks the stats: 1 fire, 2 lightning, 3 magic, 4 cold (with a length), 5 poison (minimum
//     and maximum times ten, length times two), 6 life steal, 7 mana steal, 8 stamina steal, 9 stun (length
//     only), 10 a random one of 1..5 (length 25 when the column is 0), 0xb freeze.

// ElementKind is the type byte the exe gives an El#Type token.
type ElementKind int

// Element kinds.
const (
	ElemNone ElementKind = iota
	ElemFire
	ElemLightning
	ElemMagic
	ElemCold
	ElemPoison
	ElemLifeSteal
	ElemManaSteal
	ElemStaminaSteal
	ElemStun
	ElemRandom
	ElemFreeze
)

// ParseElementKind maps the El#Type token of monstats.txt (fire, ltng, mag, cold, pois, life, mana, stam,
// stun, rand, frze) to its kind; unknown tokens give ElemNone.
func ParseElementKind(tok string) ElementKind {
	switch tok {
	case "fire":
		return ElemFire
	case "ltng":
		return ElemLightning
	case "mag":
		return ElemMagic
	case "cold":
		return ElemCold
	case "pois":
		return ElemPoison
	case "life":
		return ElemLifeSteal
	case "mana":
		return ElemManaSteal
	case "stam":
		return ElemStaminaSteal
	case "stun":
		return ElemStun
	case "rand":
		return ElemRandom
	case "frze":
		return ElemFreeze
	}

	return ElemNone
}

// Damaging reports whether the kind is plain elemental damage the engine can apply at once (fire, lightning,
// magic, cold, poison).
func (k ElementKind) Damaging() bool { return k >= ElemFire && k <= ElemPoison }

// Element is one scaled El slot.
type Element struct {
	Kind     ElementKind
	Pct      int // chance in percent, 100 or more = always
	Min, Max int
	Length   int // frames, raw
}

// ScaleElement scales one El slot: dm is the monlvl DM value of the monster's level and difficulty.
func ScaleElement(kind ElementKind, pct, minPct, maxPct, dur, dm int, noRatio bool) Element {
	e := Element{Kind: kind, Pct: pct, Min: minPct, Max: maxPct, Length: dur}

	if !noRatio {
		e.Min, e.Max = MulDiv(dm, minPct, 100), MulDiv(dm, maxPct, 100)
	}

	if e.Max < e.Min {
		e.Max = e.Min
	}

	return e
}

// Fires reports whether the slot applies to an attack: a zero chance never, 100 or more always, else a
// percent roll (roll(n) returns [0,n)) must come out below the chance.
func (e Element) Fires(roll func(n int) int) bool {
	switch {
	case e.Kind == ElemNone || e.Pct <= 0:
		return false
	case e.Pct >= 100:
		return true
	}

	return roll(100) < e.Pct
}

// PoisonTotal is the damage of a poison slot spread over its length: the exe stores the damage rate as
// ten times the scaled value (in 1/256 per frame) and the length as twice the column, so the total is
// rate*length/256. The unit of the rate is UNVERIFIED against the running game.
func (e Element) PoisonTotal() (min, max int) {
	frames := e.Length * 2

	return e.Min * 10 * frames >> 8, e.Max * 10 * frames >> 8
}
