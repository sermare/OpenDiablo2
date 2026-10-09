package d2monstats

// Player-count scaling and the classic-mode adjustment.
//
// VERIFIED 0x00571760 (called from 0x00571af0): the spawn code gets
// {hpPct, xpPct, ...players} from the player count n (clamped to at least 1).
// n < 9 indexes two identical 9-entry tables at 0x006e2888 / 0x006e28ac:
// 0,0,50,100,150,200,250,300,350. n >= 9: hp = (n-2)*50, xp = (n*5+0x82)*2.
// Classes whose monstats Align (+0x4c) is nonzero get 0%/0% and players = 1.
// HP = (min+roll) + MulDiv(min+roll, hpPct, 100), capped 0x7fffff;
// XP = xp + MulDiv(xp, xpPct, 100).
// The count (0x005331a0) is the number of players found by iterating the
// game's player table with predicate 0x00552234; in game types 1..3
// (game+0x6a) it is max(count, forced /players value at 0x0087bdb0, set to
// 0..8 by the setter at 0x00533190).

var playerPct = [9]int{0, 0, 50, 100, 150, 200, 250, 300, 350}

// PlayerBonus returns the HP and XP percentage bonuses for n players and the
// player count recorded on the monster. align is the class' Align column.
func PlayerBonus(n, align int) (hpPct, xpPct, players int) {
	if n < 1 {
		n = 1
	}

	if align != 0 {
		return 0, 0, 1
	}

	if n < len(playerPct) {
		return playerPct[n], playerPct[n], n
	}

	return (n - 2) * 50, (n*5 + 0x82) * 2, n
}

// EffectivePlayers is the count the exe feeds to PlayerBonus: in game types
// 1..3 the forced /players value (0..8) wins when it is larger.
func EffectivePlayers(actual, forced, gameType int) int {
	if gameType >= 1 && gameType <= 3 && forced > actual {
		return forced
	}

	return actual
}

// Options selects the optional behaviours of ScaleOpts.
type Options struct {
	// Players is the number of players in the game (0 means 1).
	Players int
	// Classic enables the 0x0063ff30 adjustment, which the exe applies when
	// game+0x70 is zero (no area level source; identity of that field as
	// "classic game" is inferred, not proven), difficulty > Normal and the
	// class Align is not 1.
	Classic bool
}

// classicRatio is the table at 0x006ebfa0: numerator, denominator per
// difficulty for HP (stat 7), AC (0x1f) and XP (0xd).
var classicRatio = struct{ HP, AC, XP [numDiff][2]int }{
	HP: [numDiff][2]int{Nightmare: {1, 2}, Hell: {1, 2}},
	AC: [numDiff][2]int{Nightmare: {10, 12}, Hell: {10, 12}},
	XP: [numDiff][2]int{Nightmare: {10, 17}, Hell: {10, 26}},
}

// applyClassic is VERIFIED 0x0063ff30 (table 0x006ebfa0), run after the
// player bonus: HP256 x1/2, AC x10/12, XP x10/17 (N) or x10/26 (H), all
// truncating MulDiv, and the level becomes monstats Level (normal column) +
// 25*diff. Stats.HP is recomputed from HP256 (floor).
func (s *Stats) applyClassic(c *Class, diff int) {
	r := classicRatio
	s.HP256 = MulDiv(s.HP256, r.HP[diff][0], r.HP[diff][1])
	s.HP = s.HP256 >> 8
	s.AC = MulDiv(s.AC, r.AC[diff][0], r.AC[diff][1])
	s.XP = MulDiv(s.XP, r.XP[diff][0], r.XP[diff][1])
	s.Level = c.Level[Normal] + 25*diff
}

// LevelExclusion selects which class flag keeps a class on its monstats Level
// in Nightmare/Hell. The exe tests flag bit 6 (mask at 0x006cf268). By the
// monstats flag table order (noRatio=2 and interact=9 both check out against
// the code) bit 6 is primeevil; boss would be bit 5. UNVERIFIED which column
// the bit comes from, so it is switchable.
type LevelExclusion int

// Exclusion choices.
const (
	ExcludePrimeEvil LevelExclusion = iota
	ExcludeBoss
)

// LevelExclusionFlag is the active choice.
var LevelExclusionFlag = ExcludePrimeEvil

func (c *Class) excluded() bool {
	if LevelExclusionFlag == ExcludeBoss {
		return c.Boss
	}

	return c.PrimeEvil
}
