package d2player

// statReplenishQuantity is item_replenish_quantity (ItemStatCost id 253): the "replenishes quantity" property of
// javelins such as Titan's Revenge, which win back thrown weapons while the hero fights.
const statReplenishQuantity = 253

// replenishSeconds is the time it takes a thrown weapon with the property to regain one piece, for the
// property's value. UNVERIFIED against the exe: it is taken to work like "repairs durability" (one in
// 100/value seconds); a value of 30 gives one javelin every 3.3 seconds.
func replenishSeconds(value int64) float64 {
	if value <= 0 {
		return 0
	}

	return 100 / float64(value)
}

// advanceAmmo gives a thrown weapon with "replenishes quantity" its pieces back, up to the stack limit of the
// item (or the quantity it was loaded with when that is higher). Without it the stack of a Javazon runs out
// after a few dozen throws (the Act 2 playthrough of the Amazon: Lightning Fury refused for lack of javelins).
func (g *GameControls) advanceAmmo(elapsed float64) {
	s := g.ammoStack()
	if s == nil || g.hero == nil || g.hero.Stats == nil || g.hero.Stats.Health <= 0 {
		return
	}

	var value int64

	for _, p := range s.StatItem().Props {
		if p.ID == statReplenishQuantity {
			value += p.Value
		}
	}

	every := replenishSeconds(value)
	if every <= 0 {
		return
	}

	if g.ammoCap == 0 {
		g.Infof("AMMO %s replenishes quantity (%d): one piece every %.1fs, stack %d of at most %d", s.GetItemCode(),
			value, every, s.Quantity(), s.StackLimit())
	}

	if g.ammoCap < s.Quantity() {
		g.ammoCap = s.Quantity()
	}

	if limit := s.StackLimit(); limit > g.ammoCap {
		g.ammoCap = limit
	}

	if s.Quantity() >= g.ammoCap {
		g.ammoAcc = 0

		return
	}

	for g.ammoAcc += elapsed; g.ammoAcc >= every && s.Quantity() < g.ammoCap; g.ammoAcc -= every {
		s.SetQuantity(s.Quantity() + 1)
	}
}
