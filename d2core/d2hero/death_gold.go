package d2hero

// DeathGoldCapLevel is the hero level above which the gold penalty percent stops growing.
const DeathGoldCapLevel = 20

// GoldPenalty is what a death does to the gold.
type GoldPenalty struct {
	Lost       int // gold that is gone for good
	NewCarried int // carried gold after the death
	NewStash   int // stash gold after the death
	Dropped    int // gold that lies on the floor as piles
}

// DeathGoldPenalty is PLAYER_ApplyDeathGoldPenalty 0x5331e0 (VERIFIED for single player / open games): the loss is
// min(level,20) percent of carried + stash gold. Killed by a monster the loss vanishes, the rest of the carried gold
// is dropped and nothing stays carried; when the loss exceeds the carried gold the shortfall comes out of the stash.
// Killed by a player (pk) the victim only loses that amount, which is dropped (the killer may take it), and
// keeps the remaining carried gold. The realm game type 3 has extra rules that are not modelled.
func DeathGoldPenalty(level, carried, stash int, pk bool) GoldPenalty {
	if carried < 0 {
		carried = 0
	}

	if stash < 0 {
		stash = 0
	}

	l := level
	if l > DeathGoldCapLevel {
		l = DeathGoldCapLevel
	}

	loss := l * (carried + stash) / 100
	out := GoldPenalty{Lost: loss, NewStash: stash}

	if carried < loss {
		out.NewStash = stash - (loss - carried)
		if out.NewStash < 0 {
			out.NewStash = 0
		}

		out.Lost = carried + stash - out.NewStash

		if pk {
			out.Dropped = loss
		}

		return out
	}

	if pk {
		out.Dropped = loss
		out.NewCarried = carried - loss

		return out
	}

	out.Dropped = carried - loss

	return out
}
