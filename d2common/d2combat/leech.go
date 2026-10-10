package d2combat

// Port of the leech transfer 0x57a3b0 (called from ApplyDamageToUnit 0x57a650),
// verified against the real function in the emulator (golden leech_golden.json).
// Observations only. Unverified: helper 0x646620 (called with a source file name
// and a line number, treated as a no-op) and the meaning of the sound event ids.

// Event ids passed to 0x622020 by the leech transfer.
const (
	LeechEventA = 0x97
	LeechEventB = 0x98
)

// LeechIn is the input of ApplyLeech.
type LeechIn struct {
	// Struct fields (damage struct +8, +0x38, +0x3c, +0x40).
	Total, LifeLeech, ManaLeech, StamLeech int32

	HasAttacker  bool
	AttackerKind int // unit type: 0 player, 1 monster
	Special      bool

	// Attacker pools (8.8) and flags.
	Life, MaxLife, Mana, MaxMana int32
	AttackerDead                 bool // helper 0x552230
	AttackerNoHeal               bool // state 0x5c

	DefenderIsMonster bool
	// DefenderDrain is the monstats percent byte for the difficulty (+0xa0+diff);
	// ignored for other defenders (100 is used).
	DefenderDrain   int32
	DefenderMana    int32 // 8.8
	DefenderStamina int32 // 8.8

	// DiffLifeDiv and DiffManaDiv are the divisors of the difficulty row (+0x28, +0x2c); 0 means none.
	DiffLifeDiv, DiffManaDiv int32
}

// LeechOut is the result.
type LeechOut struct {
	LifeLeech, ManaLeech int32 // struct fields afterwards
	Life, Mana           int32 // attacker pools afterwards
	Events               []int // ids passed to 0x622020 (attacker)
}

func div64(v int32) int32 { return v / 64 }

// ApplyLeech transfers life/mana to the attacker. The Roll(2) consumed after a
// combined mana + life leech is on the attacker's generator.
func ApplyLeech(r Roller, in LeechIn) LeechOut {
	out := LeechOut{LifeLeech: in.LifeLeech, ManaLeech: in.ManaLeech, Life: in.Life, Mana: in.Mana}
	if in.LifeLeech == 0 && in.ManaLeech == 0 {
		return out
	}

	p := int32(100)

	if in.DefenderIsMonster {
		p = in.DefenderDrain
		if p <= 0 {
			return out
		}
	}

	scalep := func(v int32) int32 {
		if p != 100 {
			return MulDiv(v, p, 100)
		}

		return v
	}

	l, m := in.LifeLeech<<6, in.ManaLeech<<6
	out.LifeLeech, out.ManaLeech = l, m

	playerPath := in.HasAttacker && (in.AttackerKind == 0 || (in.AttackerKind == 1 && in.Special))

	if in.HasAttacker && in.AttackerKind == 0 {
		if in.DiffLifeDiv != 0 {
			l /= in.DiffLifeDiv
			out.LifeLeech = l
		}

		if in.DiffManaDiv != 0 {
			m /= in.DiffManaDiv
			out.ManaLeech = m
		}
	}

	if !playerPath {
		// monster style leech: sum of life, mana and stamina capped by what the
		// defender has, heals the attacker's life only
		l, m = div64(l), div64(m)
		out.LifeLeech, out.ManaLeech = l, m

		life := l
		if life >= in.Total {
			life = in.Total
		}

		mana := m
		if mana >= in.DefenderMana {
			mana = in.DefenderMana
		}

		stam := in.StamLeech
		if stam >= in.DefenderStamina {
			stam = in.DefenderStamina
		}

		sum := life + mana + stam
		if sum == 0 {
			return out
		}

		sum = scalep(sum)

		if room := in.MaxLife - in.Life; sum >= room {
			sum = room
		}

		if sum <= 0 {
			return out
		}

		out.Life += sum
		out.Events = append(out.Events, LeechEventA)

		return out
	}

	if in.Total <= 0 {
		return out
	}

	used := false

	if m != 0 {
		v := div64(scalep(MulDiv(in.Total, m, 100)))
		if v > 0 {
			out.Mana = addCapped(out.Mana, v, in.MaxMana)
		}

		used = true
	}

	if l == 0 {
		if used {
			out.Events = append(out.Events, LeechEventB)
		}

		return out
	}

	if v := div64(scalep(MulDiv(in.Total, l, 100))); v > 0 && !in.AttackerDead && !in.AttackerNoHeal {
		out.Life = addCapped(out.Life, v, in.MaxLife)
	}

	if !used || r.Roll(2) == 0 {
		out.Events = append(out.Events, LeechEventA)
	} else {
		out.Events = append(out.Events, LeechEventB)
	}

	return out
}

func addCapped(cur, v, maxV int32) int32 {
	n := cur + v
	if n >= maxV {
		n = maxV
	}

	return n
}
