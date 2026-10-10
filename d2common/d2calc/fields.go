package d2calc

// skillCodes are the field codes of skillcalc.txt (rows 0..72, the id the game
// compiles them to is the row index). Verified against the shipped table.
var skillCodes = map[string]bool{
	"ln12": true,
	"dm12": true,
	"ln34": true,
	"dm34": true,
	"ln56": true,
	"dm56": true,
	"ln78": true,
	"dm78": true,
	"par1": true,
	"par2": true,
	"par3": true,
	"par4": true,
	"par5": true,
	"par6": true,
	"par7": true,
	"par8": true,
	"lvl":  true,
	"edmn": true,
	"edmx": true,
	"edln": true,
	"toht": true,
	"mana": true,
	"mps":  true,
	"math": true,
	"madm": true,
	"macr": true,
	"m1en": true,
	"m1ex": true,
	"m1el": true,
	"m2en": true,
	"m2ex": true,
	"m2el": true,
	"m3en": true,
	"m3ex": true,
	"m3el": true,
	"m1rn": true,
	"m2rn": true,
	"m3rn": true,
	"edns": true,
	"edxs": true,
	"ulvl": true,
	"blvl": true,
	"usmc": true,
	"m1eo": true,
	"m1ey": true,
	"m2eo": true,
	"m2ey": true,
	"me3o": true,
	"me3y": true,
	"enma": true,
	"exma": true,
	"edma": true,
	"enms": true,
	"exms": true,
	"len":  true,
	"clc1": true,
	"clc2": true,
	"clc3": true,
	"clc4": true,
	"rng":  true,
	"ast1": true,
	"ast2": true,
	"ast3": true,
	"ast4": true,
	"ast5": true,
	"ast6": true,
	"pst1": true,
	"pst2": true,
	"pst3": true,
	"pst4": true,
	"pst5": true,
	"pets": true,
	"skpt": true,
}

// missileCodes are the field codes of misscalc.txt.
var missileCodes = map[string]bool{
	"par1": true,
	"par2": true,
	"par3": true,
	"par4": true,
	"par5": true,
	"cpa1": true,
	"cpa2": true,
	"cpa3": true,
	"cpa4": true,
	"cpa5": true,
	"hpa1": true,
	"hpa2": true,
	"hpa3": true,
	"chp1": true,
	"chp2": true,
	"chp3": true,
	"dpa1": true,
	"dpa2": true,
	"lvl":  true,
	"edmn": true,
	"edmx": true,
	"edln": true,
	"edns": true,
	"edxs": true,
	"damn": true,
	"damx": true,
	"dmns": true,
	"dmxs": true,
	"rang": true,
	"sl12": true,
	"sd12": true,
	"sl34": true,
	"sd34": true,
	"cl12": true,
	"cd12": true,
	"cl34": true,
	"cd34": true,
	"shl1": true,
	"shd1": true,
	"chl1": true,
	"chd1": true,
	"dl12": true,
	"dd12": true,
}

// LN is the lnXY field: ParamX + (lvl-1)*ParamY, 0 for lvl < 1. Verified: the
// txt description says "a+lvl*b" but the code uses lvl-1.
func LN(a, b, lvl int) int {
	if lvl < 1 {
		return 0
	}

	return a + (lvl-1)*b
}

// DM is the dmXY field (SKILL_DiminishingReturns, 0x646ed0, verified against
// the real game with the emulator oracle):
//
//	t = (110*lvl)/(lvl+6)          (integer division FIRST)
//	r = a + t*(b-a)/100            (truncating)
//	r = min(r, b)                  (the game caps at the second parameter even
//	                                when b < a, so a decreasing pair gives b)
//
// The game returns 0 for lvl < 1 in the dm12..dm56 fields (dm78 has no such
// guard, see DM78).
func DM(a, b, lvl int) int {
	if lvl < 1 {
		return 0
	}

	return DM78(a, b, lvl)
}

// DM78 is DM without the lvl < 1 guard, which is how the game's dm78 field
// (case 7 of 0x6477d0) calls the shared routine. lvl+6 == 0 would fault in the
// game; here 0 is returned.
func DM78(a, b, lvl int) int {
	if lvl+6 == 0 {
		return 0
	}

	t := int32(110*lvl) / int32(lvl+6)
	r := int(t)*(b-a)/100 + a

	if r > b {
		r = b
	}

	return r
}

// IsFieldCode reports whether code (first four lower case characters of an
// identifier) is a known field code of the kind.
func IsFieldCode(code string, kind Kind) bool {
	if kind == KindMissile {
		return missileCodes[code]
	}

	return skillCodes[code]
}
