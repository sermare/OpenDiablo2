package d2drop

const (
	maxAffixLevel  = 99
	maxAffixPerSet = 3 // up to 3 prefixes and 3 suffixes on one item

	// rareJewelBase is the constant added to rand(2) for the number of
	// affixes on a rare jewel. The call is RAND_RollSeedRangeWithBase(2) with
	// the base in a register the decompiler dropped: UNVERIFIED (3 matches
	// the known 3-4 magic affixes on a rare jewel).
	rareJewelBase = 3
)

// rareAffixCounts is the table at 6e45a8 (VERIFIED by reading the binary),
// indexed by a generator step & 7: the number of magic affixes of a rare.
var rareAffixCounts = [8]int{3, 4, 4, 5, 5, 5, 6, 6}

// Affix is a MagicPrefix/MagicSuffix row.
type Affix struct {
	ID        string
	Prefix    bool
	Version   int
	Spawnable bool
	Rare      bool // may appear on rare and crafted items
	Level     int
	MaxLevel  int // 0 = no maximum
	Frequency int
	Group     int
	Class     string // class restriction, "" for none
	IType     []string
	EType     []string
}

// AffixItem describes the item an affix is being rolled for.
type AffixItem struct {
	ILvl       int
	QLvl       int
	MagicLevel int
	Types      []string // item type and all of its ancestors
	Class      string   // class the item belongs to, "" for none
	Quality    Quality
	Version    int // 100 for Lord of Destruction items
	Jewel      bool
}

// AffixLevel is the affix level (alvl) of an item (VERIFIED, 5bf1c0): with no
// magic level it is ilvl - qlvl/2, or 2*ilvl - 99 once ilvl reaches
// 99 - qlvl/2; with a magic level it is ilvl + magiclvl. Clamped to 1..99.
func AffixLevel(ilvl, qlvl, magicLevel int) int {
	var alvl int

	switch {
	case magicLevel != 0:
		alvl = ilvl + magicLevel
	case ilvl < maxAffixLevel-qlvl/2:
		alvl = ilvl - qlvl/2
	default:
		alvl = 2*ilvl - maxAffixLevel
	}

	return minInt(maxInt(alvl, 1), maxAffixLevel)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}

	return b
}

func anyIn(list, types []string) bool {
	for _, a := range list {
		for _, b := range types {
			if a != "" && a == b {
				return true
			}
		}
	}

	return false
}

// Eligible reports whether the affix can spawn on the item at all, ignoring
// the groups already on it.
func (a *Affix) Eligible(it *AffixItem) bool {
	alvl := AffixLevel(it.ILvl, it.QLvl, it.MagicLevel)

	switch {
	case !a.Spawnable, a.Version > it.Version:
		return false
	case alvl < a.Level, a.MaxLevel != 0 && alvl > a.MaxLevel:
		return false
	case (it.Quality == QualityRare || it.Quality == QualityCrafted) && !a.Rare:
		return false
	case a.Class != "" && a.Class != it.Class:
		return false
	case !anyIn(a.IType, it.Types), anyIn(a.EType, it.Types):
		return false
	}

	return true
}

func (a *Affix) weight(it *AffixItem) int {
	if it.MagicLevel != 0 {
		return a.Frequency * a.Level
	}

	return a.Frequency
}

// PickAffix is ITEMGEN_PickAffixLod. Unless force is set it first passes a 50%
// gate. Candidates are the eligible affixes whose group is not in
// usedGroups (VERIFIED: weight = frequency, or frequency*level when the item
// has a magic level; roll = rand(sum+1) walked by subtraction, so the last
// candidate effectively gets one extra weight). It returns nil when the gate
// fails or there is no candidate.
//
// The game retries a clashing group up to 251 times instead of filtering the
// candidates; the distribution is the same but the number of generator steps
// consumed differs.
func PickAffix(rng RNG, pool []Affix, it *AffixItem, usedGroups map[int]bool, force bool) *Affix {
	if !force && !rng.Chance() {
		return nil
	}

	cands := make([]*Affix, 0, len(pool))
	sum := 0

	for i := range pool {
		a := &pool[i]
		if !a.Eligible(it) || (a.Group != 0 && usedGroups[a.Group]) {
			continue
		}

		cands = append(cands, a)
		sum += a.weight(it)
	}

	if len(cands) == 0 {
		return nil
	}

	roll := int(rng.Roll(int32(sum + 1)))

	for _, a := range cands {
		w := a.weight(it)
		if roll < w {
			return a
		}

		roll -= w
	}

	return cands[len(cands)-1]
}

// MagicAffixes is the result of a magic or rare affix roll.
type MagicAffixes struct {
	Prefixes, Suffixes []*Affix
}

// RollMagicAffixes is ITEMGEN_RollMagicAffixes (quality 4): a prefix behind
// the 50% gate, then a suffix behind the gate, except that the suffix is
// forced when no prefix was found, so a magic item always has an affix.
func RollMagicAffixes(rng RNG, prefixes, suffixes []Affix, it *AffixItem) MagicAffixes {
	var res MagicAffixes

	used := map[int]bool{}

	pre := PickAffix(rng, prefixes, it, used, false)
	if pre != nil {
		res.Prefixes = append(res.Prefixes, pre)
		used[pre.Group] = true
	}

	if suf := PickAffix(rng, suffixes, it, used, pre == nil); suf != nil {
		res.Suffixes = append(res.Suffixes, suf)
	}

	return res
}

// RareAffixCount rolls how many magic affixes a rare item gets (VERIFIED
// table; the jewel base is UNVERIFIED, see rareJewelBase).
func RareAffixCount(rng RNG, jewel bool) int {
	if jewel {
		return rareJewelBase + int(rng.Roll(2))
	}

	return rareAffixCounts[rng.Roll(8)] // power of two: a masked step
}

// RollRareAffixes is the affix part of ITEMGEN_RollRareAffixesLod (VERIFIED
// against the decompilation, 5bf8c0): n successful picks, each on the side
// chosen by a coin flip unless one side is exhausted (3 picked, or no
// candidate left); picks are forced (no 50% gate); a failed pick marks the
// side exhausted and does not count. The two rare names are not rolled here,
// see PickRareName.
func RollRareAffixes(rng RNG, prefixes, suffixes []Affix, it *AffixItem) MagicAffixes {
	var res MagicAffixes

	used := map[int]bool{}
	n := RareAffixCount(rng, it.Jewel)
	preDone, sufDone := false, false

	for done := 0; done < n; {
		suffix := false

		switch {
		case preDone && sufDone:
			return res
		case preDone:
			suffix = true
		case sufDone:
		default:
			suffix = rng.Chance()
		}

		pool, count := prefixes, len(res.Prefixes)
		if suffix {
			pool, count = suffixes, len(res.Suffixes)
		}

		a := PickAffix(rng, pool, it, used, true)
		if a == nil {
			if suffix {
				sufDone = true
			} else {
				preDone = true
			}

			continue
		}

		used[a.Group] = true

		if suffix {
			res.Suffixes = append(res.Suffixes, a)
		} else {
			res.Prefixes = append(res.Prefixes, a)
		}

		if count+1 >= maxAffixPerSet {
			if suffix {
				sufDone = true
			} else {
				preDone = true
			}
		}

		done++
	}

	return res
}

// PickRareName picks one of n rare prefix/suffix names uniformly
// (ITEMGEN_PickRareNameLod). The caller filters the names by item type.
// It returns -1 if n is zero.
func PickRareName(rng RNG, n int) int {
	if n < 1 {
		return -1
	}

	return int(rng.Roll(int32(n)))
}
