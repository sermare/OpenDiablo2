package d2combat

// Effect trace of the hit reaction 0x57ae50, verified against the real function
// in the emulator (golden react_golden.json: every call the function makes to
// the animation / mode / state helpers is recorded). Observations only.
//
// Effect names and arguments (all numbers are the ones the exe passes):
//
//	setstate 0x5c 1        defender with state 0x36 died
//	monpre                 helper 0x5a2f20, always first for monsters
//	mondeath 1             monster death (0x57ac20)
//	playmon M, anim        monster plays mode M (0xd knockback, 6 block, 3 hit)
//	setmode 0x13           mode 0x13 set via 0x571300 (no animation)
//	roomflag               0x64d280 + unit flag 0x8000
//	notify                 0x5a1e80
//	stagger V              result of 0x57aa60 (only evaluated when state 0x15 is absent)
//	plmode M               player mode via 0x57ea70 (0xd dodge/avoid, 0 death, 0x13 knockback)
//	plmode9 M              player mode via 0x57e9b0 (9 block, 4 hit recovery)
//	setstat 0x5f F         last block frame
//	x5513c0 0xc            evade effect
type Effect struct {
	Name string
	Args []int
}

// ReactionIn is the input of ReactionEffects.
type ReactionIn struct {
	DefenderType int // unit type: 0 player, 1 monster
	ClassID      int
	Mode         int // defender mode (+0x10)
	Result       uint32
	Frame        int32 // game frame (+0xa8)
	LastBlock    int32 // defender stat 0x5f
	FasterBlock  int32 // defender stat 0x66
	HPBarStat    int32 // defender stat 0x160
	HPBarCalc    uint8 // helper 0x5a3070
	TotalPos     bool  // damage struct total > 0

	State36, State15, State41, State42, State44 bool

	SkillRef   bool // helper 0x644e00 finds the dodge/avoid/evade skill
	SkillData  bool // helper 0x457c20 data has the evade effect (word +0xfc > 0)
	HasKnock   bool // monster has mode 0xd
	HasBlock   bool // monster has mode 6
	HasHit     bool // monster has mode 3 (only used through Stagger)
	Stagger    func() bool
	staggerSet bool
}

// ReactionOut is the effect list and the (possibly rewritten) result word.
type ReactionOut struct {
	Effects []Effect
	Result  uint32
}

// ReactionEffects reproduces the sequence of effects of 0x57ae50.
func ReactionEffects(in ReactionIn) ReactionOut {
	var out ReactionOut

	res := in.Result
	out.Result = res

	add := func(name string, args ...int) { out.Effects = append(out.Effects, Effect{name, args}) }

	if in.State36 {
		if res&2 != 0 {
			add("setstate", 0x5c, 1)
		}

		return out
	}

	stagger := func() bool {
		v := in.Stagger()
		b := 0
		if v {
			b = 1
		}

		add("stagger", b)

		return v
	}

	switch in.DefenderType {
	case 1:
		if in.Mode == 0 || in.Mode == 0xc {
			return out
		}

		if res&8 != 0 && !in.HasKnock {
			res = res&^0x8 | 0x4
			out.Result = res
		}

		add("monpre")

		switch {
		case res&2 != 0:
			add("mondeath", 1)
		case res&8 != 0:
			add("playmon", 0xd)
			add("anim")
		case res&0x10 != 0:
			if res&0x4000 == 0 && in.ClassID != 0xf3 && in.ClassID != 0x14d && in.ClassID != 0x2c1 && in.HasBlock {
				add("playmon", 6)
				add("anim")
			} else {
				add("setmode", 0x13)
			}
		case res&4 != 0:
			if !in.State15 && stagger() {
				add("roomflag")
				add("setmode", 0x13)
				add("notify")

				return out
			}

			add("playmon", 3)
			add("anim")
			add("notify")
		case res&0x4000 != 0:
			add("roomflag")
			add("setmode", 0x13)
			add("notify")
		case res&1 != 0 && in.TotalPos:
			d := int(in.HPBarStat&0xff) - int(in.HPBarCalc)
			if d < 0 {
				d = -d
			}

			if d > 4 {
				add("roomflag")
			}
		}
	case 0:
		in.playerEffects(&out, add, stagger)
	}

	return out
}

func (in ReactionIn) playerEffects(out *ReactionOut, add func(string, ...int), stagger func() bool) {
	res := in.Result

	switch {
	case res&0x80 != 0 || res&0x100 != 0:
		st := in.State42
		if res&0x80 != 0 {
			st = in.State41
		}

		if st && in.SkillRef {
			add("plmode", 0xd)
		}
	case res&0x200 != 0:
		if in.State44 && in.SkillRef && in.SkillData {
			add("x5513c0", 0xc)
		}
	case res&0x8010 != 0:
		if res&0x4000 != 0 {
			return
		}

		if in.Frame-in.LastBlock > FasterBlockDivisor(in.FasterBlock)+15 {
			add("plmode9", 9)
			add("setstat", 0x5f, int(in.Frame))
		}
	case res&2 != 0:
		if in.Mode != 0 && in.Mode != 0x11 {
			add("plmode", 0)
		}
	case res&8 != 0:
		add("plmode", 0x13)
	case res&4 != 0:
		if !in.State15 && stagger() {
			add("roomflag")

			return
		}

		add("plmode9", 4)
	case res&0x4000 != 0:
		add("roomflag")
	}
}

// FasterBlockDivisor is fbr/8 truncated toward zero (the exe adds 7 to negative values).
func FasterBlockDivisor(fbr int32) int32 { return fbr / 8 }

// StaggerGate is helper 0x57aa60 with the generator wired in: step advances the
// defender's seed and returns the new low word (the exe masks it with 1 and 3,
// verified in 0x46de50: result = newLow & (n-1)). A coin is only drawn when the
// hit is below the respective threshold. True means "no hit recovery".
func StaggerGate(step func() uint32, frozen bool, poisonDmg, total int32, hitClass int, maxLife int32, isMonster, monsterHasHit bool) bool {
	if frozen || (poisonDmg != 0 && poisonDmg == total) || total < 0x100 {
		return true
	}

	d := int32(0x10)

	switch hitClass {
	case 2, 6, 10, 11:
		d = 8
	case 4, 8:
		d = 0x20
	case 5:
		d = 0x40
	}

	if maxLife/d > total {
		return true
	}

	if maxLife/(d/2) > total && step()&1 == 0 {
		return true
	}

	if maxLife/(d/4) > total && step()&3 == 0 {
		return true
	}

	return isMonster && !monsterHasHit
}
