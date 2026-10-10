package d2combat

// The melee hit pipeline from the damage struct to hit point changes, verified
// end to end against Game.exe in the emulator (golden e2e_golden.json): attacker
// struct (0x5794e0, BuildAttackerDamage), scale and resists (0x579d70 +
// 0x579ef0 + 0x579e50, ResolveStruct) and apply (0x57a650 with the struct flag
// argument 0 as 0x57b4b0 calls it, ApplyHit). Observations only.

// Descriptor table 0x72ff38 (12 rows). Offsets are struct byte offsets; the
// stats are unit stat ids, -1 meaning none.
type resistRow struct {
	off                         int
	resist, maxStat, pierce     int
	absorbPct, absorbFlat, slot int
	skipUnlessMonster           bool // rows 9..11 (leeches): only processed for plain monster attackers
	scale                       bool // multiplied by the damage scale percent first
}

var resistRows = [12]resistRow{
	{8, 36, -1, -1, -1, -1, 1, false, true},
	{16, 39, 40, 333, 142, 143, 2, false, true},
	{28, 41, 42, 334, 144, 145, 2, false, true},
	{36, 43, 44, 335, 148, 149, 2, false, true},
	{32, 37, 38, -1, 146, 147, 2, false, true},
	{48, 43, 44, 335, -1, -1, 0, false, false},
	{52, 43, 44, 335, -1, -1, 0, false, false},
	{44, 110, -1, 336, -1, -1, 0, false, false},
	{40, 45, 46, 336, -1, -1, 0, false, true},
	{56, -1, -1, -1, -1, -1, 0, true, true},
	{60, -1, -1, -1, -1, -1, 0, true, true},
	{64, -1, -1, -1, -1, -1, 0, true, true},
}

// word returns a pointer to dword i of the struct (the C layout).
func (d *Damage) word(i int) *int32 {
	switch i {
	case 2:
		return &d.Physical
	case 3:
		return &d.DamagePct
	case 4:
		return &d.Fire
	case 5:
		return &d.Burn
	case 6:
		return &d.BurnLen
	case 7:
		return &d.Lightning
	case 8:
		return &d.Magic
	case 9:
		return &d.Cold
	case 10:
		return &d.Poison
	case 11:
		return &d.PoisonLen
	case 12:
		return &d.ColdLen
	case 13:
		return &d.FreezeLen
	case 14:
		return &d.LifeLeech
	case 15:
		return &d.ManaLeech
	case 16:
		return &d.StaminaLeech
	case 17:
		return &d.StunLen
	case 18:
		return &d.Heal
	case 19:
		return &d.Total
	case 20, 21, 22, 23:
		return &d.Unknown20[i-20]
	case 24:
		return &d.HitClass
	}

	panic("d2combat: no such damage struct word")
}

// ScaleIn are the classification results 0x579d70 uses (helpers 0x63fed0,
// 0x63fa40, 0x63fe00, 0x44d580 with 0x80000000).
type ScaleIn struct {
	HaveBoth                         bool // attacker and defender present and different
	AttackerKind, DefenderKind       int
	AttackerSpecial, DefenderSpecial bool
	AttackerFlag80, DefenderFlag80   bool  // monster flag 0x80000000
	DefenderBoss                     bool  // 0x63fa40(0, defender)
	AttackerMinion                   bool  // 0x63fe00
	RowBossPct                       int32 // difficulty row +0x38; <= 0 means 100 (null pointer in the exe means 100)
	RowPresent                       bool
}

// DamageScalePercent is 0x579d70: the percent applied to every scalable struct
// field before the resists. 100 means no scaling.
func DamageScalePercent(in ScaleIn) int32 {
	if !in.HaveBoth {
		return 100
	}

	if in.DefenderKind == 0 {
		if in.AttackerKind == 0 || in.AttackerSpecial || in.AttackerFlag80 {
			return 0x11
		}

		return 100
	}

	if in.DefenderSpecial && in.AttackerSpecial {
		return 0x19
	}

	if in.AttackerSpecial && in.DefenderBoss {
		if !in.RowPresent {
			return 100
		}

		return in.RowBossPct
	}

	if in.AttackerMinion && in.DefenderFlag80 {
		if in.DefenderSpecial {
			return 200
		}

		return 400
	}

	return 100
}

// ResolveIn is the input of ResolveStruct.
type ResolveIn struct {
	Dmg Damage
	// Stats of attacker and defender.
	AGet, DGet func(stat int) int32
	// Classification.
	AttackerKind, DefenderKind       int
	AttackerPlainMonster             bool // attacker is a monster and not special
	DefenderPlainMonster             bool // defender is a monster and not special
	DefenderUndead, DefenderDemon    bool
	AttackerState2F                  bool
	DefenderState85, DefenderState83 bool
	Scale                            ScaleIn
	Expansion                        bool
	Difficulty                       int
	Penalty                          int // difficulty row first dword (LoD)
}

// ResolveStruct applies the scale percent, the freeze/stun tweaks, resists, flat
// reductions and absorb to the damage struct and sums Total (0x579ef0).
func ResolveStruct(in ResolveIn) Damage {
	d := in.Dmg
	pct := DamageScalePercent(in.Scale)

	if pct != 100 {
		for _, row := range resistRows {
			if !row.scale {
				continue
			}

			p := d.word(row.off / 4)
			if *p > 0 {
				*p = MulDiv(*p, pct, 100)
			}
		}
	}

	// 0x579e50
	if d.ColdLen > 0 || d.FreezeLen > 0 {
		if in.DGet(0x99) != 0 {
			d.ColdLen, d.FreezeLen = 0, 0
		} else if in.DGet(0x76) != 0 {
			d.ColdLen /= 2
			d.FreezeLen /= 2
		}
	}

	if d.PoisonLen > 0 && in.DefenderState85 {
		d.PoisonLen = 0
	}

	if d.BurnLen > 0 && in.DefenderState83 {
		d.BurnLen = 0
	}

	// ignore flag: defender undead / demon / other class vs struct flags
	ign := false

	if in.DefenderKind == 1 {
		if in.DefenderUndead && d.Flags&DamageFlagIgnoreUndead != 0 {
			ign = true
		}

		if in.DefenderDemon && d.Flags&DamageFlagIgnoreDemon != 0 {
			ign = true
		}

		if !in.DefenderUndead && !in.DefenderDemon && d.Flags&DamageFlagIgnoreBeast != 0 {
			ign = true
		}
	} else if d.Flags&DamageFlagIgnoreBeast != 0 {
		ign = true
	}

	var flat [3]int32

	f1 := in.DGet(0x22) << 8
	if f1 > 0 && d.Unknown20[1] > 0 {
		f1 = MulDiv(f1, d.Unknown20[1], 0x400)
	}

	f2 := in.DGet(0x23) << 8
	if f2 > 0 && d.Unknown20[1] > 0 {
		f2 = MulDiv(f2, d.Unknown20[1], 0x400)
	}

	flat[1], flat[2] = f1, f2

	for _, row := range resistRows {
		if row.skipUnlessMonster && !in.AttackerPlainMonster {
			break
		}

		p := d.word(row.off / 4)
		if *p <= 0 {
			*p = 0

			continue
		}

		rin := ResistInput{IsPhysical: row.resist == 36, NoDifficultyPenalty: row.resist == 36 || row.resist == 37, Ignore: in.DefenderPlainMonster}

		if row.resist != -1 {
			rin.Resist = int(in.DGet(row.resist))
		}

		if row.pierce != -1 {
			rin.HasPierce, rin.Pierce = true, int(in.AGet(row.pierce))
		}

		if row.maxStat != -1 {
			rin.HasMaxResist, rin.MaxResistBonus = true, int(in.DGet(row.maxStat))
		}

		rin.ZeroPhysical = in.AttackerState2F && in.DefenderUndead

		if in.Expansion {
			rin.DifficultyPenalty = in.Penalty
		} else {
			rin.DifficultyPenalty = ClassicResistPenalty(in.Difficulty)
		}

		var ap, af int32

		if row.absorbPct != -1 {
			ap = in.DGet(row.absorbPct)
			af = in.DGet(row.absorbFlat)
		}

		out, heal := ResolveComponent(*p, flat[row.slot], EffectiveResist(rin), ign, row.absorbPct != -1, ap, af)
		*p = out
		d.Heal += heal
	}

	d.Total = d.Physical + d.Fire + d.Lightning + d.Magic + d.Cold + d.Poison

	if in.AttackerPlainMonster {
		d.Total += d.LifeLeech
	}

	return d
}

// ApplyIn is the input of ApplyHit (0x57a650 with the struct flag argument 0,
// struct flags 0x20 and result bit 1 set, as 0x57b4b0 calls it).
type ApplyIn struct {
	Dmg Damage
	// Attacker.
	AttackerKind                 int
	AttackerSpecial              bool
	Life, MaxLife, Mana, MaxMana int32
	AttackerDead, AttackerNoHeal bool
	// Defender.
	DefenderKind                             int
	DefLife, DefMaxLife, DefMana, DefStamina int32
	DefDead, DefNoHeal                       bool
	DefenderIsMonster                        bool
	// DefenderNoDamage is helper 0x452b20 (monster class flag 0xf clear): a monster
	// defender takes no damage at all. DefDead (0x552230) returns early too.
	DefenderNoDamage         bool
	DefenderDrain            int32
	DiffLifeDiv, DiffManaDiv int32
}

// StateCall is one call of the timed state appliers made by ApplyHit.
type StateCall struct {
	Name string
	Args []int32
}

// ApplyOut is the result of ApplyHit.
type ApplyOut struct {
	Dmg                          Damage
	DefLife, DefMana, DefStamina int32
	Life, Mana                   int32
	States                       []StateCall
	Events                       []int
}

// ApplyHit mirrors the hit point part of 0x57a650: the physical component is
// capped at the defender's life, leech is transferred, the absorbed heal and the
// total are applied, mana and stamina are burned and the timed states are
// requested (always, in the order stun, chill, freeze, poison, burn).
func ApplyHit(r Roller, in ApplyIn) ApplyOut {
	d := in.Dmg
	o := ApplyOut{Dmg: d, DefLife: in.DefLife, DefMana: in.DefMana, DefStamina: in.DefStamina, Life: in.Life, Mana: in.Mana}

	if in.DefDead || (in.DefenderKind == 1 && in.DefenderNoDamage) {
		return o
	}

	if d.Physical >= in.DefLife {
		d.Physical = in.DefLife
	}

	lo := ApplyLeech(r, LeechIn{
		Total: d.Physical, LifeLeech: d.LifeLeech, ManaLeech: d.ManaLeech, StamLeech: d.StaminaLeech,
		HasAttacker: true, AttackerKind: in.AttackerKind, Special: in.AttackerSpecial,
		Life: in.Life, MaxLife: in.MaxLife, Mana: in.Mana, MaxMana: in.MaxMana, AttackerDead: in.AttackerDead, AttackerNoHeal: in.AttackerNoHeal,
		DefenderIsMonster: in.DefenderIsMonster, DefenderDrain: in.DefenderDrain, DefenderMana: in.DefMana, DefenderStamina: in.DefStamina,
		DiffLifeDiv: in.DiffLifeDiv, DiffManaDiv: in.DiffManaDiv,
	})
	d.LifeLeech, d.ManaLeech = lo.LifeLeech, lo.ManaLeech
	o.Life, o.Mana = lo.Life, lo.Mana
	o.Events = append(o.Events, lo.Events...)

	// defender heal from absorb (0x5786e0)
	if v := d.Heal; v > 0 && !in.DefDead && !in.DefNoHeal {
		o.DefLife += v
		if o.DefLife >= in.DefMaxLife {
			o.DefLife = in.DefMaxLife
		}
	}

	if d.Total > 0 {
		l := o.DefLife - d.Total
		if l < 0x100 {
			l = 0
		}

		o.DefLife = l
	}

	if v := d.ManaLeech; v > 0 {
		m := o.DefMana - v
		if m < 0x100 {
			m = 0
		}

		o.DefMana = m
	}

	if v := d.StaminaLeech; v > 0 {
		s := o.DefStamina - v
		if s < 0x100 {
			s = 0
		}

		o.DefStamina = s
	}

	o.States = []StateCall{
		{"stun", []int32{d.StunLen}},
		{"chill", []int32{d.ColdLen}},
		{"freeze", []int32{d.FreezeLen}},
		{"poison", []int32{d.Poison, d.PoisonLen}},
		{"burn", []int32{d.Burn, d.BurnLen}},
	}

	if o.DefLife > 0 {
		d.Result &^= ResultDied
	} else {
		d.Result |= ResultDied
	}

	// (kill item events 0xa and 9 follow here in the exe; they go through the
	// event dispatcher 0x5be860 and are not modelled)

	o.Dmg = d

	return o
}
