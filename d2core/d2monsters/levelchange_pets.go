package d2monsters

import (
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
