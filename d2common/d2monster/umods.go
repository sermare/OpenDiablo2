package d2monster

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"

// ModCandidate is one monumod.txt row the modifier roll may pick: its id and
// the cpick / upick weight of the game's difficulty.
type ModCandidate struct {
	ID     int
	Weight int
}

// MaxUniqueMods is the number of modifier ids a monster remembers (the exe's
// copy of the leader's modifiers to a minion handles up to 9, VERIFIED name
// MONSTER_CopyLeaderUModsToMinion 0x59e4c0).
const MaxUniqueMods = 9

// PickUniqueMods draws n distinct modifier ids by weight (the shape of
// MONSTER_PickRandomBossUMod 0x59e0e0 / MONSTER_PickRandomNonBossUMod 0x59e1e0:
// weighted pick of a row not yet chosen). The pick weights are the monumod
// cpick / upick columns; how many are drawn per rank and difficulty, and which
// rows are filtered, are UNVERIFIED (the callers decide n and the candidates).
func PickUniqueMods(r *d2rand.Seed, cands []ModCandidate, n int) []int {
	pool := make([]ModCandidate, 0, len(cands))

	for _, c := range cands {
		if c.Weight > 0 {
			pool = append(pool, c)
		}
	}

	var out []int

	for len(out) < n && len(out) < MaxUniqueMods && len(pool) > 0 {
		total := 0
		for _, c := range pool {
			total += c.Weight
		}

		at := roll(r, total)
		for i, c := range pool {
			if at < c.Weight {
				out = append(out, c.ID)
				pool = append(pool[:i], pool[i+1:]...)

				break
			}

			at -= c.Weight
		}
	}

	return out
}
