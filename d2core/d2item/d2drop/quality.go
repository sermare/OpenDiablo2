package d2drop

const (
	// qualityFloor is the constant 128 every quality test compares against:
	// a test succeeds when rand(chance-adj) < 128, so the success probability
	// is 128/(chance-adj) and is capped at 1.
	qualityFloor = 128

	mfDiminishUnique = 250
	mfDiminishSet    = 500
	mfDiminishRare   = 600

	// mfThreshold is the first x (100+MF) that is diminished.
	mfThreshold = 111
)

// QualityInput is everything the quality roll of a dropped base item needs.
type QualityInput struct {
	ILvl      int // item level, taken from the dropper
	QLvl      int // base item level
	MagicFind int // killer's item_magicbonus (stat 0x50, owner's included)
	Mods      QualityMods

	TypeNormal bool // ItemTypes.Normal
	TypeMagic  bool // ItemTypes.Magic
	TypeRare   bool // ItemTypes.Rare
	Unique     bool // base item flagged unique
	Quest      bool // base item flagged quest
}

// DiminishMagicFind is ITEMGEN_ApplyMagicFindDiminishing (VERIFIED constants):
// x is 100+MF; below 111 it is returned unchanged, otherwise
// (x-100)*factor/((x-100)+factor)+100.
func DiminishMagicFind(x, factor int) int {
	if x < mfThreshold {
		return x
	}

	mf := x - 100

	return mf*factor/(mf+factor) + 100
}

// RollQuality is ITEMGEN_RollDropQuality (556690). VERIFIED against the
// decompilation: the early outs, the unique/set/rare/magic tests (chance,
// MF scaling and diminishing factors 250/500/600, minimum clamp, reduction by
// the treasure class modifier in 1/1024, the 128 floor) and the order of
// the tests.
//
// The whole function, tail included, is checked roll for roll against the
// real game running under an emulator (testdata/quality.json): the modulus of
// the superior and normal tests is the tested value, a roll below 128 in the
// last test gives normal quality and anything else low quality.
func RollQuality(rng RNG, r *Ratio, in QualityInput) Quality {
	if in.TypeNormal {
		return QualityNormal
	}

	if in.Unique || (in.TypeMagic && in.Quest) {
		return QualityUnique
	}

	d := in.ILvl - in.QLvl
	mf := in.MagicFind

	// MF < -99 skips the unique/set/rare/magic tests entirely.
	if mf >= -99 {
		if hit(rng, tierChance(r.Unique, d, mf, mfDiminishUnique), in.Mods.Unique) {
			return QualityUnique
		}

		if hit(rng, tierChance(r.Set, d, mf, mfDiminishSet), in.Mods.Set) {
			return QualitySet
		}

		if in.TypeRare &&
			hit(rng, tierChance(r.Rare, d, mf, mfDiminishRare), in.Mods.Rare) {
			return QualityRare
		}

		// An always-magic type is magic without a test.
		if in.TypeMagic {
			return QualityMagic
		}

		if hit(rng, magicChance(r.Magic, d, mf), in.Mods.Magic) {
			return QualityMagic
		}
	}

	return rollTail(rng, r, d)
}

func rollTail(rng RNG, r *Ratio, d int) Quality {
	hq := (r.HiQuality.Base - div(d, r.HiQuality.Divisor)) * qualityFloor
	if hq <= 0 || rng.Roll(int32(hq)) < qualityFloor {
		return QualitySuperior
	}

	normal := (r.Normal.Base - div(d, r.Normal.Divisor)) * qualityFloor
	if normal <= 0 {
		return QualityNormal
	}

	// VERIFIED (5569e9): rand < 128 gives normal, otherwise low quality.
	if rng.Roll(int32(normal)) < qualityFloor {
		return QualityNormal
	}

	return QualityLow
}

// div is C integer division, 0 for a zero divisor (the game's tables never
// have one).
func div(a, b int) int {
	if b == 0 {
		return 0
	}

	return a / b
}

// tierChance is the denominator of a unique/set/rare test before the
// treasure class modifier: (base - d/divisor)*128, or with magic find
// (base - d/divisor)*12800/diminished(100+MF), never below the table minimum.
func tierChance(dr DropRatio, d, mf, factor int) int {
	c := (dr.Base - div(d, dr.Divisor)) * qualityFloor

	if mf != 0 {
		if eff := DiminishMagicFind(100+mf, factor); eff != 0 {
			c = (dr.Base - div(d, dr.Divisor)) * 12800 / eff
		}
	}

	return maxInt(c, dr.Min)
}

// magicChance is the same without the diminishing returns on magic find.
func magicChance(dr DropRatio, d, mf int) int {
	c := (dr.Base - div(d, dr.Divisor)) * qualityFloor

	if mf != 0 {
		c = (dr.Base - div(d, dr.Divisor)) * 12800 / (mf + 100)
	}

	return maxInt(c, dr.Min)
}

// hit reports whether a quality test succeeds: the denominator is reduced by
// mod/1024 of itself, and the test passes outright if nothing is left,
// otherwise when rand(rest) < 128.
func hit(rng RNG, chance, mod int) bool {
	adj := mod * chance / 1024 // (mod*chance)>>10, mod*chance >= 0

	if chance == adj || chance-adj < 0 {
		return true
	}

	return rng.Roll(int32(chance-adj)) < qualityFloor
}

// FallbackQuality is the quality the game falls back to when no row of the
// requested quality can be picked (ITEMGEN_ApplyQualityToItem): unique ->
// rare, set -> magic, rare -> magic, magic -> superior.
func FallbackQuality(q Quality) Quality {
	switch q {
	case QualityUnique:
		return QualityRare
	case QualitySet, QualityRare:
		return QualityMagic
	case QualityMagic:
		return QualitySuperior
	default:
		return q
	}
}
