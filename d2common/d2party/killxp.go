package d2party

// SplitKillXP is the experience split of 0x0057c6b0, called by the kill credit
// routine 0x0057c990 when the credited player is in a party. The base xp is
// the monster's experience BEFORE any per-recipient scaling. levels holds the
// character level of each recipient. VERIFIED from the disassembly:
//   - one recipient: no split, the base xp stands;
//   - otherwise total = xp + (n-1)*xp*89/256 (a 34.8% bonus per extra member);
//   - recipient i gets trunc(total/sum(levels) * level_i), the quotient taken
//     as a 32-bit float;
//   - the caller then scales each share with the recipient's own level
//     difference (d2herostats.KillXP).
//
// Which members are recipients: alive members within squared distance 6400
// (RecipientRadiusSq) of the killer, at most 8. The unit of the position
// (tile or subtile) and the exact reference unit are UNVERIFIED.
func SplitKillXP(xp int, levels []int) []int {
	n := len(levels)
	out := make([]int, n)

	sum := 0
	for _, l := range levels {
		sum += l
	}

	if n == 0 || sum <= 0 {
		return out
	}

	if n == 1 {
		out[0] = xp

		return out
	}

	total := xp + (n-1)*xp*0x59>>8
	per := float32(total) / float32(sum)

	for i, l := range levels {
		out[i] = int(float64(l) * float64(per))
	}

	return out
}

// RecipientRadiusSq is the squared distance limit for a party member to take
// part in a kill's experience (0x0057c5a0 compares against 0x1900).
const RecipientRadiusSq = 0x1900

// MaxRecipients is the size of the recipient array in the original (an
// assertion fires at 8 entries).
const MaxRecipients = 8
