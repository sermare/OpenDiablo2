package d2object

// Opening of a loot container: the part of OBJECT_OperateLootContainer (0x583e70) that decides how often
// and with which forced item quality the chest treasure class is rolled. VERIFIED from the disassembly.
// Each roll of the class (ITEMGEN_DropChestTreasure 0x583a60) takes a forced quality: 0 lets the game roll
// it, 4 is magic, 5 set, 6 rare, 7 unique. What the container looks at is the first item made and whether
// it is of magical quality (ITEM_IsMagicalQuality 0x62a290: quality 4 to 9).

// SparklyChestIndex is the objects.txt row of the sparkly chest, the container with its own tiers.
const SparklyChestIndex = 397

// Forced qualities used by the containers.
const (
	QualityAuto   = 0
	QualityMagic  = 4
	QualitySet    = 5
	QualityRare   = 6
	QualityUnique = 7
)

// VariantRareChance is the percent of variant containers that force rare instead of magic.
const VariantRareChance = 5

// IsMagicalQuality is ITEM_IsMagicalQuality: quality ids 4 to 9.
func IsMagicalQuality(q int) bool { return q > 3 && q < 10 }

// LootSink is what a container opening acts on.
type LootSink interface {
	// Drop rolls the chest treasure class once with a forced quality (QualityAuto for none). made says
	// whether an item came out, magical whether the first one is of magical quality.
	Drop(forced int) (made, magical bool)
	// Create makes n items of a base code at the container ("gld " gold, "hp3 ", "mp3 " potions).
	Create(code string, n int)
}

// OpenGeneric is the plain container branch. The variant bit costs one extra seed step that decides the
// forced quality of every roll: magic, or rare in 5 percent. An unlocked non variant container opens
// empty when the next percent roll is 24 or less. A locked one rolls twice. A variant container that made
// no magical item rolls up to ten more times until one is magical.
func OpenGeneric(r Rand, s LootSink, locked, variant bool) {
	forced := QualityAuto

	if variant {
		forced = QualityMagic
		if r.Roll(100) < VariantRareChance {
			forced = QualityRare
		}
	}

	if r.Roll(100) <= EmptyChestThreshold && !variant && !locked {
		return
	}

	rolls := 1
	if locked {
		rolls = 2
	}

	magical := 0

	for i := 0; i < rolls; i++ {
		if made, mag := s.Drop(forced); made && mag {
			magical++
		}
	}

	if !variant || magical > 0 {
		return
	}

	for i := 0; i < 10; i++ {
		if made, mag := s.Drop(forced); made && mag {
			return
		}
	}
}

// SparklyTier names the branch a sparkly chest takes for a roll in [0, 10000).
type SparklyTier int

// Sparkly chest tiers, by the roll boundaries 200, 600, 1200, 3200, 6200.
const (
	SparklyUnique     SparklyTier = iota // below 200: a unique, twice if not magical
	SparklySet                           // below 600
	SparklyRare                          // below 1200
	SparklyMagicThree                    // below 3200: up to ten magic rolls until three are magical
	SparklyMagicTwo                      // below 6200: up to ten until two, gold fills the rest
	SparklyJackpot                       // 6200 and above
)

// SparklyTierOf maps a roll in [0, 10000) to its tier.
func SparklyTierOf(roll int) SparklyTier {
	switch {
	case roll < 200:
		return SparklyUnique
	case roll < 600:
		return SparklySet
	case roll < 1200:
		return SparklyRare
	case roll < 3200:
		return SparklyMagicThree
	case roll < 6200:
		return SparklyMagicTwo
	}

	return SparklyJackpot
}

// OpenSparkly is the sparkly chest. The variant roll still costs a seed step but is not used. A forced
// quality drop that makes nothing, or whose second try is not magical, falls through to the jackpot, as the
// original's control flow does.
func OpenSparkly(r Rand, s LootSink) {
	r.Roll(100) // the variant step is taken before the object id is compared; its result is unused here

	switch SparklyTierOf(r.Roll(10000)) {
	case SparklyUnique:
		sparkleForced(s, QualityUnique)
	case SparklySet:
		sparkleForced(s, QualitySet)
	case SparklyRare:
		sparkleForced(s, QualityRare)
	case SparklyMagicThree:
		made, magical := 0, 0

		for i := 0; i < 10; i++ {
			if m, mag := s.Drop(QualityMagic); m {
				made++

				if mag {
					magical++
				}
			}

			if magical >= 3 {
				break
			}
		}

		if made == 0 {
			sparkleJackpot(s)
		}
	case SparklyMagicTwo:
		plain, magical := 0, 0

		for i := 0; i < 10; i++ {
			if m, mag := s.Drop(QualityMagic); m {
				if mag {
					magical++
				} else {
					plain++
				}
			}

			if magical >= 2 {
				break
			}
		}

		if plain == 0 {
			if m, _ := s.Drop(QualityAuto); m {
				plain = 1
			}
		} else if plain >= 7 {
			return
		}

		if gold := 7 - plain; gold > 0 {
			s.Create("gld ", gold)
		}
	default:
		sparkleJackpot(s)
	}
}

// sparkleForced is the three single quality tiers: one forced drop, a second when the first is not
// magical, and the jackpot unless one of them is magical.
func sparkleForced(s LootSink, quality int) {
	for try := 0; try < 2; try++ {
		made, mag := s.Drop(quality)
		if made && mag {
			return
		}

		if !made && try == 0 {
			break
		}
	}

	sparkleJackpot(s)
}

// sparkleJackpot: up to ten magic rolls stopping at the first magical item, topped up with unforced
// rolls to four drops when fewer non magical ones were made, then five gold piles, two greater healing and
// two greater mana potions.
func sparkleJackpot(s LootSink) {
	plain := 0

	for i := 0; i < 10; i++ {
		if made, mag := s.Drop(QualityMagic); made {
			if mag {
				break
			}

			plain++
		}
	}

	for ; plain < 4; plain++ {
		s.Drop(QualityAuto)
	}

	s.Create("gld ", 5)
	s.Create("hp3 ", 2)
	s.Create("mp3 ", 2)
}
