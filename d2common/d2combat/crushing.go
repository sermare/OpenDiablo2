package d2combat

// Item event ids passed to the callbacks (edx). Event 6 is the missile hit.
const EventMissile = 6

// CrushingInput describes a crushing blow roll (callback 0x5bdbf0, item stat
// 136). Life values are 8.8.
type CrushingInput struct {
	Chance        int   // attacker stat 136
	DefenderKind  int   // unit type: 0 player, 1 monster, other
	Special       bool  // helper 0x63fed0 (classes 0x10f,0x152,0x167,0x230,0x231)
	Boss          bool  // helper 0x63fa40 (monstats flag 0x40)
	DataFlag2     bool  // monster data +0x16 & 2
	PlayerCount   int   // defender stat 100 (monster_playercount)
	Event         int   // item event id
	Life          int32 // defender life (stat 6)
	PhysResistRaw int   // defender stat 36, unclamped
}

// CrushingOutcome is the result of RollCrushingBlow.
type CrushingOutcome struct {
	Rolled  bool  // the chance roll was made (one RNG step)
	Hit     bool  // callback returned 1
	Removed int32 // life removed (8.8), after physical resist
	Life    int32 // defender life after, floored at 0
	Died    bool  // result bit 2 must be set on the damage struct
	Knock   bool  // helper 0x622020(def,0x93) called (removed > 0)
}

// PlayerCountBonus is helper 0x571720: percent added to a monster's crushing
// divisor by the number of players. Verified: [0,0,50,100,150,200,250,300,350]
// for n < 9, else (n-2)*50 (clamped input >= 1 by the caller).
func PlayerCountBonus(n int) int {
	if n < 9 {
		if n < 0 {
			n = 0
		}

		return []int{0, 0, 50, 100, 150, 200, 250, 300, 350}[n]
	}

	return (n - 2) * 50
}

// CrushingDivisor is the divisor of current life removed by a crushing blow.
// Verified in 0x5bdbf0: player defenders and special classes 10, bosses and
// data-flag-2 monsters 8, other monsters 4 (+ the player count bonus except
// for the special case), other unit types 4; the missile event doubles it.
func CrushingDivisor(kind int, special, boss, flag2 bool, playerCount, event int) int32 {
	d := int32(4)

	switch {
	case kind == 0:
		d = 10
	case kind == 1:
		switch {
		case special:
			d = 10
		default:
			if boss || flag2 {
				d = 8
			}

			pc := playerCount
			if pc < 1 {
				pc = 1
			}

			if b := PlayerCountBonus(pc); b != 0 {
				d += MulDiv(d, int32(b), 100)
			}
		}
	}

	if event == EventMissile {
		d *= 2
	}

	return d
}

// RollCrushingBlow performs the chance roll (Roll(100) < Chance, skipped when
// Chance <= 0) and computes the life removed. Verified against the exe.
func RollCrushingBlow(r Roller, in CrushingInput) CrushingOutcome {
	var o CrushingOutcome

	o.Life = in.Life

	if in.Chance <= 0 {
		return o
	}

	o.Rolled = true

	if int(r.Roll(100)) >= in.Chance {
		return o
	}

	o.Hit = true
	d := CrushingDivisor(in.DefenderKind, in.Special, in.Boss, in.DataFlag2, in.PlayerCount, in.Event)
	removed := in.Life / d

	if p := in.PhysResistRaw; p > 0 {
		if p > 100 {
			p = 100
		}

		removed -= MulDiv(removed, int32(p), 100)
	}

	life := in.Life - removed
	if life <= 0 {
		life = 0
		o.Died = true
	}

	o.Removed = removed
	o.Life = life
	o.Knock = removed > 0

	return o
}

// OpenWoundsBase is helper 0x5bd9c0: the level scaled damage per tick before
// the +40 offset. t holds the five per-level slopes {9,18,27,36,45}; the
// breakpoints are 15, 30, 45, 60. Verified.
func OpenWoundsBase(level int, t [5]int) int {
	if level <= 1 {
		return 0
	}

	cum := 0
	prev := 1

	for i, bp := range []int{15, 30, 45, 60} {
		if level <= bp {
			return cum + (level-prev)*t[i]
		}

		cum += (bp - prev) * t[i]
		prev = bp
	}

	return cum + (level-prev)*t[4]
}

// OpenWoundsInput describes an open wounds roll (callback 0x5bda80, stat 135).
type OpenWoundsInput struct {
	Chance        int  // attacker stat 135
	AttackerLevel int  // attacker stat 12
	DefenderKind  int  // 0 player, 1 monster
	DefenderFlagC bool // monster data flag tested with index 12 (+0x16 & 0xc)
	Event         int
}

// OpenWoundsOutcome: when Hit, a 200 frame state 0x3e with stat 0x4a (hp
// regen) = -Value is created on the defender.
type OpenWoundsOutcome struct {
	Rolled bool
	Hit    bool
	Value  int // positive; the stat is set to -Value
}

// OpenWoundsLength is the frame length of the open wounds state (verified).
const OpenWoundsLength = 200

// RollOpenWounds rolls the chance (Roll(100) < Chance, skipped when Chance <=
// 0) and derives the damage value. Verified against the exe.
func RollOpenWounds(r Roller, in OpenWoundsInput) OpenWoundsOutcome {
	var o OpenWoundsOutcome

	if in.Chance <= 0 {
		return o
	}

	o.Rolled = true

	if int(r.Roll(100)) >= in.Chance {
		return o
	}

	lvl := in.AttackerLevel
	if lvl < 1 {
		lvl = 1
	}

	v := OpenWoundsBase(lvl, [5]int{9, 18, 27, 36, 45}) + 40

	switch in.DefenderKind {
	case 0:
		v /= 4
		if in.Event == EventMissile {
			v /= 2
		}
	case 1:
		if in.DefenderFlagC {
			v /= 2
		}
	}

	o.Hit = true
	o.Value = v

	return o
}
