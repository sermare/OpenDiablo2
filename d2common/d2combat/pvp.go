package d2combat

// Player versus player. The damage scale is the pre-step 0x579d70 of
// COMBAT_ApplyResistsToDamageStruct (see CombatantScalePercent): when the
// defender is a player and the attacker is player controlled the damage is
// scaled to 17 percent (VERIFIED, the binary spot-check of the skills-combat
// notes); a plain monster attacker keeps 100.

// PvPPercent is the scale between two heroes.
func PvPPercent() int {
	return CombatantScalePercent(false, true, true, true, true)
}

// PvPDamage scales a hero's raw hit for a hero defender.
func PvPDamage(raw int) int {
	if raw <= 0 {
		return 0
	}

	return int(ScaleByPercent(int32(raw), PvPPercent()))
}

// PvPReceive is what a defending hero loses from a scaled hit: the flat damage
// reduction first, then the physical resistance percent (the order of the two
// is VERIFIED at 0x579c90: flat reduction first, then the percent; the 17 percent scale runs before both). It never heals.
func PvPReceive(scaled, physResist, reduce int) int {
	d := ApplyResist(scaled-reduce, physResist)
	if d < 0 {
		return 0
	}

	return d
}

// PvPKillGivesEar reports whether the death of a hero killed by another hero
// (or by a pet or mercenary led by another hero) leaves an ear on the ground
// at the victim's feet. VERIFIED w44 (Game.exe 0x57efc4 caller, 0x558160): the
// death handler drops the ear whenever the victim is a player and the killer is
// a different player or a monster led by a different player; neither the
// handler nor the item builder (0x557cd0, 0x556e50) tests hardcore. The ear
// carries the victim's current level (stat 0xc) and name; only the item's
// 0x8000 flag (ladder) is copied from the victim's character. The flag
// argument stays for callers and is ignored.
func PvPKillGivesEar(victimHardcore bool) bool { return true }

// PvPParts is a skill hit on a hero by damage type, in whole points.
type PvPParts struct{ Phys, Fire, Light, Magic, Cold int }

// Total is the sum of the components.
func (p PvPParts) Total() int { return p.Phys + p.Fire + p.Light + p.Magic + p.Cold }

// Scale applies the PvP percent to every component (integer division, as
// ScaleByPercent does).
func (p PvPParts) Scale(pct int) PvPParts {
	f := func(v int) int {
		if v <= 0 {
			return 0
		}

		return int(ScaleByPercent(int32(v), pct))
	}

	return PvPParts{f(p.Phys), f(p.Fire), f(p.Light), f(p.Magic), f(p.Cold)}
}

// Slice is the wire form: phys, fire, lightning, magic, cold.
func (p PvPParts) Slice() []int { return []int{p.Phys, p.Fire, p.Light, p.Magic, p.Cold} }

// PvPPartsFromSlice is the inverse of Slice; ok is false for a wrong length.
func PvPPartsFromSlice(s []int) (p PvPParts, ok bool) {
	if len(s) != 5 {
		return p, false
	}

	return PvPParts{s[0], s[1], s[2], s[3], s[4]}, true
}

// PvPDefender is what a defending hero resists (percent, flat reductions).
type PvPDefender struct {
	PhysResist, MagicResist, FireResist, ColdResist, LightResist int
	Reduce, MagicReduce                                          int
	// FlatRate is the DamageRate (missile stat 0x147) of the missile that hit: above 0 the flat reductions count
	// only FlatRate / 1024 of their value per hit (VERIFIED 0x579ef0, wave 31). 0 keeps them whole.
	FlatRate int
}

// PvPReceiveFlat is PvPReceive for one damage type with the flat reduction of the type: the exe's flat reduction
// is stat 34 for physical and stat 35 for fire, lightning, cold and magic (VERIFIED descriptor table 0x72ff38,
// slot per type); with a DamageRate above 0 it is scaled by rate / 1024 in 8.8 before it is subtracted
// (0x579ef0, 0x579c90), so a standing fire (Fire Wall rate 41) pays about one 25th of it per hit.
func PvPReceiveFlat(scaled, resist, flat, rate int) int {
	if rate <= 0 {
		return PvPReceive(scaled, resist, flat)
	}

	if scaled <= 0 {
		return 0
	}

	f := ScaleFlatReduction(flat, int32(rate))

	out, _ := ReduceComponent(scaled<<FixedShift, f, resist, false, false, 0, 0)
	if out <= 0 {
		return 0
	}

	// the exe keeps life in 8.8; the whole-point API rounds to the nearest point (UNVERIFIED rounding; engine choice, the exe never rounds a stored value to whole points)
	return (out + 1<<(FixedShift-1)) >> FixedShift
}

// PvPReceiveParts is what a defending hero loses from a scaled skill hit:
// physical takes the flat reduction then the physical resist (as PvPReceive),
// magic its flat reduction and resist, the elements their resist. Order of
// reduction and resist: flat first, then the percent (VERIFIED 0x579c90). Fire, cold and lightning take the
// magic flat reduction (stat 35) like magic does (wave 31, VERIFIED slot table 0x72ff38; before they took none).
func PvPReceiveParts(p PvPParts, d PvPDefender) int {
	total := PvPReceiveFlat(p.Phys, d.PhysResist, d.Reduce, d.FlatRate)
	total += PvPReceiveFlat(p.Magic, d.MagicResist, d.MagicReduce, d.FlatRate)
	total += PvPReceiveFlat(p.Fire, d.FireResist, d.MagicReduce, d.FlatRate)
	total += PvPReceiveFlat(p.Cold, d.ColdResist, d.MagicReduce, d.FlatRate)
	total += PvPReceiveFlat(p.Light, d.LightResist, d.MagicReduce, d.FlatRate)

	return total
}

// PvPCarry keeps the fraction of a point that the PvP scale cuts off a hit, per
// damage type, for one defender. Life is 8.8 fixed point in the exe, so a
// burning ground tick of 4.88 fire at 17 percent takes 0.83 life and the ticks
// add up; scaling every tick to whole points first throws all of it away (the
// 5 point ticks of Meteor's burning ground scaled to 0). The remainder of one
// hit is added to the next hit on the same defender.
type PvPCarry [5]int

// Scale scales a hit given in 8.8 fixed point per type (physical, fire,
// lightning, magic, cold) to whole points, carrying the fraction over. The
// result never exceeds the running total of the exact scaled damage.
func (c *PvPCarry) Scale(raw [5]int32, pct int) PvPParts {
	var out [5]int

	for i, v := range raw {
		if v <= 0 {
			continue
		}

		sum := int(v)*pct/100 + c[i]
		out[i], c[i] = sum>>8, sum&0xff
	}

	return PvPParts{out[0], out[1], out[2], out[3], out[4]}
}

// PvPScaledBounds is the exact range of whole points a defender has been charged after n hits whose 8.8 fixed point
// values add up to sum88 in one damage type, when every hit went through PvPCarry.Scale at pct percent (starting
// from an empty carry). Each hit adds floor(v*pct/100) units of 1/256 point to the carry, the carry pays out whole
// points and keeps a remainder under one point, so with t the sum of those floors:
//
//	sum88*pct/100 - n < t <= sum88*pct/100   and   charged*256 <= t < charged*256 + 256
//
// hence hi = floor(sum88*pct/25600), never more than the exact scaled damage, and lo = the smallest charged with
// charged*25600 > sum88*pct - 100*n - 25600 (a deficit of under one point plus n/256 for the truncation).
// 9f-pvp-skills-ear checks its logged hits against the same formula.
func PvPScaledBounds(pct int, sum88 int64, n int) (lo, hi int) {
	exact := sum88 * int64(pct)
	hi = int(exact / 25600)

	num := exact - 100*int64(n) - 25600
	if num < 0 {
		return 0, hi
	}

	return int(num/25600) + 1, hi
}

// PvPDefendCarry keeps the fraction of a point that the defender's elemental
// resist cuts off a hit, per element (fire, lightning, cold), in hundredths.
// A burning ground tick of one scaled point against 75 percent resist is 0.25
// life; whole points would throw every such tick away (9f-pvp-skills-ear: the
// Meteor ground ticks were "scaled=1 taken=0").
type PvPDefendCarry [3]int

// ReceiveParts is PvPReceiveParts with the elemental fractions carried to the
// next hit on the same defender.
func (c *PvPDefendCarry) ReceiveParts(p PvPParts, d PvPDefender) int {
	total := PvPReceiveFlat(p.Phys, d.PhysResist, d.Reduce, d.FlatRate)
	total += PvPReceiveFlat(p.Magic, d.MagicResist, d.MagicReduce, d.FlatRate)

	el := [3]struct{ v, resist int }{{p.Fire, d.FireResist}, {p.Light, d.LightResist}, {p.Cold, d.ColdResist}}
	for i, e := range el {
		// the magic flat reduction (stat 35) comes off the element first, then the percent (0x579c90); in
		// hundredths of a point so that a DamageRate scaled reduction keeps its fraction
		left := flatLeft100(e.v, d.MagicReduce, d.FlatRate)
		if left <= 0 {
			continue
		}

		m := 100 - e.resist
		if m < 0 {
			m = 0
		}

		sum := left*m/100 + c[i]
		total += sum / 100
		c[i] = sum % 100
	}

	return total
}

// flatLeft100 is what remains of a whole-point element hit after the flat reduction, in hundredths of a point (the
// reduction is scaled by rate / 1024 when the rate is above 0), never below 0.
func flatLeft100(v, flat, rate int) int {
	if v <= 0 {
		return 0
	}

	f := flat * 100

	if f > 0 && rate > 0 {
		f = f * rate / 0x400
	}

	if left := v*100 - f; left > 0 {
		return left
	}

	return 0
}
