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

// PvPKillGivesEar reports whether killing a hero leaves the killer an ear.
// UNVERIFIED (public game knowledge, not the binary): ears exist only for
// hardcore kills; a softcore kill has no special drop and the dead hero
// handles its own corpse and items as for any death.
func PvPKillGivesEar(victimHardcore bool) bool { return victimHardcore }

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
}

// PvPReceiveParts is what a defending hero loses from a scaled skill hit:
// physical takes the flat reduction then the physical resist (as PvPReceive),
// magic its flat reduction and resist, the elements their resist. Order of
// reduction and resist is UNVERIFIED.
func PvPReceiveParts(p PvPParts, d PvPDefender) int {
	total := PvPReceive(p.Phys, d.PhysResist, d.Reduce)
	total += PvPReceive(p.Magic, d.MagicResist, d.MagicReduce)
	total += PvPReceive(p.Fire, d.FireResist, 0)
	total += PvPReceive(p.Cold, d.ColdResist, 0)
	total += PvPReceive(p.Light, d.LightResist, 0)

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
