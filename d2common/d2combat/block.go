package d2combat

// MaxBlockChance is the hard cap on block chance (verified, 0x4b).
const MaxBlockChance = 75

// PlayerBlockInput holds the inputs of a player's block chance.
type PlayerBlockInput struct {
	HasShield        bool // no shield -> 0
	ToBlock          int  // stat 20
	ClassBlockFactor int  // charstats BlockFactor
	Dex              int
	Level            int
	// IncludeDex selects the in-combat chance; the game passes a flag for it.
	IncludeDex bool
}

// PlayerBlockChance returns a player's block chance. Verified in
// COMBAT_GetBlockChance (0x6228d0):
//
//	c = toBlock + blockFactor
//	if includeDex { c = (dex-15)*c / (2*max(level,1)) }
//	c = min(c, 75)
//
// There is no lower clamp: a low dex gives a negative chance (which the roll
// treats as no block). Division truncates toward zero.
func PlayerBlockChance(in PlayerBlockInput) int {
	if !in.HasShield {
		return 0
	}

	c := in.ToBlock + in.ClassBlockFactor

	if in.IncludeDex {
		lvl := in.Level
		if lvl < 1 {
			lvl = 1
		}

		c = (in.Dex - 15) * c / (lvl * 2)
	}

	return capBlock(c)
}

// MonsterBlockChance returns a monster's block chance: its stat 20 if the
// monster can block (monstats +0xf bit 2), else 0, capped at 75. Verified.
func MonsterBlockChance(canBlock bool, toBlock int) int {
	if !canBlock {
		return 0
	}

	return capBlock(toBlock)
}

func capBlock(c int) int {
	if c > MaxBlockChance {
		return MaxBlockChance
	}

	return c
}

// RollShieldBlock rolls a shield block. Verified in COMBAT_RollShieldBlock
// (0x57bfa0): a chance below 1 means no block and NO step is consumed; a
// player who is moving (and not in mode 2) has the chance divided by 3
// (integer division). The meaning of that condition (running) is inferred, so
// the caller supplies it as playerMoving. Blocked iff roll < chance.
func RollShieldBlock(r Roller, chance int, playerMoving bool) (blocked bool) {
	if chance < 1 {
		return false
	}

	if playerMoving {
		chance /= 3
	}

	ok, _ := roll100(r, chance)

	return ok
}
