package d2monreg

// Creation of the monsters a room's preset (DS1) nodes ask for. FUN_0054c470
// (drlgpop.ResolveMonster) says which class a node is; this is the placement
// half of it: the exact spot first, then, when the class allows it, the ring
// search of 4 (FUN_0054c210 decides), each creation taking one step of the
// room seed.
//
// Read from the decompilation of FUN_0054c470 (no emulator golden for it).

// retryRingClasses is FUN_0054c210's table: classes 0xe5 + index in this set
// are not retried with the wider search, every other class is.
func retryAllowed(class int) bool {
	i := class - 0xe5
	if i < 0 || i > 0xa4 {
		return true
	}

	return !(i == 0 || (i >= 55 && i <= 59) || i == 163 || i == 164)
}

// Wanderer is the room seed step of FUN_0054ceb0 for a room without wandering
// monsters (the creation of wanderers is not modelled).
func (g *Game) Wanderer(room *Room) { room.Seed.Step() }

// PlacePreset creates one monster of a preset node at (x, y) (level subtiles)
// of the room. It returns nil when no spot was found.
func (g *Game) PlacePreset(w World, room *Room, class, x, y int, pop *Population) *Unit {
	flags := 0

	if class >= 0 && class < len(g.Tables.Mons) && g.Tables.Mons[class].Has(18) {
		flags = 8
	}

	p := Placement{Room: room, Class: class, X: x, Y: y, Ring: -1, Flags: uint16(flags)}

	px, py, ok := g.Tables.Place(w, p)
	if !ok && retryAllowed(class) {
		p.Ring = 4
		px, py, ok = g.Tables.Place(w, p)
	}

	if !ok {
		return nil
	}

	return g.create(pop, room, class, px, py)
}
