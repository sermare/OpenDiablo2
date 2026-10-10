package d2monsters

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2path"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2summon"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// PetFlagsOf looks a pettype name (the skill's pettype column, as stored in
// MinionOptions.Tag) up in PetType.txt. Unknown names have no flags, so the
// pet is released like the exe does for a type with neither column set.
func (d *Director) PetFlagsOf(petType string) d2summon.PetFlags {
	if rec := d.asset.Records.PetTypes[petType]; rec != nil {
		return d2summon.PetFlags{Warp: rec.Warp, Range: rec.Range}
	}

	return d2summon.PetFlags{}
}

// FaceTowards implements d2monster.Facer (GargoyleTrap turns to its target).
// The exe sets a 64-way facing byte; the engine's sprites have 8 directions,
// so the unit is turned toward the cardinal-axis point instead (the byte to
// sprite direction mapping is UNVERIFIED).
func (d *Director) FaceTowards(b *d2monster.Brain, x, y, _ int) {
	if u := d.unitOf(b); u != nil && u.m.Alive() {
		u.m.Face(float64(x), float64(y))
	}
}

// OwnerChangedLevel applies MERC_RelocatePetsWithOwner (0x5732b0, VERIFIED) to
// the owner's summoned units after the owner moved to another level of the
// same act: pets of a warp type are teleported next to the owner, pets of a
// range type farther than 40 subtiles from the owner's old position (oldX,
// oldY) are removed, and pets of other types are removed. Mercenaries are not
// in the pet list and are handled by their own follow rule. It returns the
// number of pets that followed and that were dropped.
func (d *Director) OwnerChangedLevel(owner *d2mapentity.Player, oldX, oldY int) (followed, dropped int) {
	var list []*unit

	for _, u := range d.units {
		if u.ally != nil && u.ally.owner == owner && u.merc == nil {
			list = append(list, u)
		}
	}

	for _, u := range list {
		dx, dy := u.b.X-oldX, u.b.Y-oldY

		switch d2summon.LevelChangeFate(d.PetFlagsOf(u.ally.opt.Tag), dx*dx+dy*dy) {
		case d2summon.FateFollow:
			if d.teleportNextToOwner(u) {
				followed++

				continue
			}

			d.expire(u)

			dropped++
		case d2summon.FateStay:
		case d2summon.FateDrop:
			d.expire(u)

			dropped++
		}
	}

	return followed, dropped
}

// CarriedPets are the pets that travel with their owner to the next level of
// the same act (see TakePetsForLevelChange).
type CarriedPets struct {
	units    []*unit
	oldFrame int
}

// Len is the number of pets that travel.
func (c *CarriedPets) Len() int {
	if c == nil {
		return 0
	}

	return len(c.units)
}

// TakePetsForLevelChange is the first half of the level change rule for a game
// screen that rebuilds the map (and its director) on every change: it plans
// the owner's pets with d2summon.CarryOverList (distances from the old
// position oldX, oldY), detaches the ones that follow so AdoptPets can put
// them into the next director, and releases all other pets of the owner. On
// an act change nothing follows. It reports how many follow and how many were
// released.
func (d *Director) TakePetsForLevelChange(owner *d2mapentity.Player, oldX, oldY int, actChange bool) (c *CarriedPets, followed, dropped int) {
	var list []*unit

	for _, u := range d.sortedUnits() {
		if u.ally != nil && u.ally.owner == owner && u.merc == nil {
			list = append(list, u)
		}
	}

	refs := make([]d2summon.PetRef, len(list))
	for i, u := range list {
		dx, dy := u.b.X-oldX, u.b.Y-oldY
		refs[i] = d2summon.PetRef{Flags: d.PetFlagsOf(u.ally.opt.Tag), DistSq: dx*dx + dy*dy}
	}

	travel := map[int]bool{}
	for _, i := range d2summon.CarryOverList(refs, actChange) {
		travel[i] = true
	}

	c = &CarriedPets{oldFrame: d.frame}

	for i, u := range list {
		if !travel[i] || !u.m.Alive() {
			d.expire(u)

			dropped++

			continue
		}

		d.engine.RemoveEntity(u.m)
		d.forget(u)

		c.units = append(c.units, u)
		followed++
	}

	return c, followed, dropped
}

// AdoptPets is the second half: the carried pets join this director, placed
// next to the owner. Their remaining lifetime is kept; targets and movement
// of the old level are forgotten. A pet that finds no free cell is released.
// It returns the number adopted.
func (d *Director) AdoptPets(c *CarriedPets) int {
	if c == nil {
		return 0
	}

	n := 0

	for _, u := range c.units {
		if u.ally.until > 0 {
			u.ally.until = d.frame + (u.ally.until - c.oldFrame)
		}

		u.ally.target, u.ally.strikeAt = nil, nil
		u.m.StopMoving()

		d.nextID++
		u.b.ID = d.nextID

		b := u.b
		u.m.Blocker = func(x, y int) bool { return d.fp.BlockedFor(b.ID, x, y) }

		u.mv, u.hadTarget, u.attackTarget, u.blocked = nil, false, 0, 0
		u.b.HasTarget, u.b.TargetID, u.summoner = false, 0, 0
		u.b.WakeNow(d.frame)

		d.units[u.b.ID] = u
		d.byEntity[u.m.ID()] = u
		d.engine.AddEntity(u.m)

		if !d.teleportNextToOwner(u) {
			d.expire(u)
			continue
		}

		x, y := u.m.SubtilePos()
		d.fp.Move(u.b.ID, x, y, d2path.FlagMonster)

		n++
	}

	return n
}
