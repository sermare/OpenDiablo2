package d2hireling

import "github.com/OpenDiablo2/OpenDiablo2/d2common/d2statlist"

// Gear is the effect of a mercenary's equipment on its stats.
//
// Equipped item properties go through the normal item stat code onto the merc's
// stat list (hirelings.md: "Equipment bonuses stack through the normal item stat
// code", FUN_005752a0 re-applies the equipped items' stats on revive; VERIFIED
// structure). The exact combination formulas below are modelled on the hero's
// (d2statlist.Compute) and are UNVERIFIED for the merc: the weapon's damage is
// added to the table damage, defense adds armor pieces' defense to the table
// defense, resists add the item resists to the table resist.
type Gear struct {
	Str, Dex int
	MaxHP    int
	Defense  int
	AR       int
	DmgMin   int
	DmgMax   int
	// Resist is fire, cold, lightning, poison (d2statlist.ResFire...).
	Resist [4]int
	// RawResist is the same sum before the (UNVERIFIED) cap above, and MaxResistBonus the
	// items' max resist stats: the combat code feeds both to d2combat.EffectiveResist, which
	// applies the verified difficulty penalty first and the cap after it.
	RawResist      [4]int
	MaxResistBonus [4]int
	// PhysResist and DamageReduction are the item stats 36 and 34.
	PhysResist      int
	DamageReduction int
}

// ApplyGear folds the merc's equipped items (d2statlist items with Slot 1, 3, 4, 5;
// the merc has no other slots) into its table stats. With no items the result equals
// the table stats exactly, so a merc without gear is unchanged. Broken items give
// nothing.
func ApplyGear(s Stats, items []d2statlist.Item) Gear {
	g := Gear{
		Str: s.Str, Dex: s.Dex, MaxHP: s.MaxHP, Defense: s.Defense, AR: s.AR,
		DmgMin: s.DmgMin, DmgMax: s.DmgMax,
		Resist: [4]int{s.Resist, s.Resist, s.Resist, s.Resist},
	}

	list := d2statlist.NewList()
	armor := 0

	var weapon *d2statlist.Item

	for i := range items {
		it := &items[i]
		if it.Broken || !validSlot(it.Slot) {
			continue
		}

		armor += it.DefenseOf(s.Level)

		for _, p := range it.AllProps() {
			switch p.ID {
			case d2statlist.StatArmorPct, d2statlist.StatMaxDmgPct, d2statlist.StatMinDmgPct, d2statlist.StatArmorClass:
				// item specific: handled with their item
			default:
				list.Add(p.ID, p.Param, p.Value)
			}
		}

		if it.Slot == d2statlist.SlotRightHand && it.Weapon != nil {
			weapon = it
		}
	}

	g.Str += int(list.Get(d2statlist.StatStrength))
	g.Dex += int(list.Get(d2statlist.StatDexterity))
	g.MaxHP += int(list.Get(d2statlist.StatMaxHP))
	g.MaxHP += g.MaxHP * int(list.Get(d2statlist.StatMaxHPPct)) / 100
	g.Defense += armor + int(list.Get(d2statlist.StatArmorClass))
	g.AR += int(list.Get(d2statlist.StatToHit))
	g.AR += g.AR * int(list.Get(d2statlist.StatToHitPct)) / 100

	if weapon != nil {
		wmin, wmax := weaponDamage(weapon)
		g.DmgMin += wmin
		g.DmgMax += wmax
	}

	g.DmgMin += int(list.Get(d2statlist.StatMinDamage))
	g.DmgMax += int(list.Get(d2statlist.StatMaxDamage))

	if pct := int(list.Get(d2statlist.StatDamagePct)); pct != 0 {
		g.DmgMin += g.DmgMin * pct / 100
		g.DmgMax += g.DmgMax * pct / 100
	}

	res := [4][2]int{
		{d2statlist.StatFireResist, d2statlist.StatMaxFireRes}, {d2statlist.StatColdResist, d2statlist.StatMaxColdRes},
		{d2statlist.StatLightResist, d2statlist.StatMaxLightRes}, {d2statlist.StatPoisonResist, d2statlist.StatMaxPoisonRes},
	}

	for i, ids := range res {
		g.Resist[i] += int(list.Get(ids[0]))
		g.RawResist[i] = g.Resist[i]
		g.MaxResistBonus[i] = int(list.Get(ids[1]))

		// UNVERIFIED: the usual 75 cap (+ the items' max resist bonus), no difficulty penalty
		if limit := 75 + int(list.Get(ids[1])); g.Resist[i] > limit && g.Resist[i] > s.Resist {
			g.Resist[i] = maxInt(limit, s.Resist)
		}
	}

	g.PhysResist = int(list.Get(d2statlist.StatDamageResist))
	g.DamageReduction = int(list.Get(d2statlist.StatNormalReduce))

	return g
}

// validSlot: the merc's body locations head 1, torso 3, right hand 4, left hand 5.
func validSlot(s int) bool { return s == 1 || s == 3 || s == 4 || s == 5 }

// weaponDamage is the weapon's table damage with its enhanced damage percent
// (two-handed table damage for a two-hander; ethereal +50%). Strength/dexterity
// bonuses are not applied to the merc (UNVERIFIED).
func weaponDamage(w *d2statlist.Item) (min, max int) {
	wb := w.Weapon
	min, max = wb.Min, wb.Max

	if wb.TwoHanded && wb.TwoMax > 0 {
		min, max = wb.TwoMin, wb.TwoMax
	}

	if w.Ethereal {
		min, max = d2statlist.EtherealDamage(min), d2statlist.EtherealDamage(max)
	}

	var edMin, edMax int

	for _, p := range w.AllProps() {
		switch p.ID {
		case d2statlist.StatMinDmgPct:
			edMin += int(p.Value)
		case d2statlist.StatMaxDmgPct:
			edMax += int(p.Value)
		}
	}

	return min + min*edMin/100, max + max*edMax/100
}
