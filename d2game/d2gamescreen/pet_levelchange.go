package d2gamescreen

// Pets and level changes. The game screen drops the monster director with the
// map on every level change, which would silently release every summon. The
// exe's rule (MERC_RelocatePetsWithOwner 0x5732b0, VERIFIED; see
// d2summon.LevelChangeFate) keeps pets of a warp type with the owner inside an
// act. takePets runs before the map is rebuilt, adoptPets after the hero
// stands in the new level; an act change carries nothing over.

// takePets detaches the hero's pets that follow (and releases the others) from
// the director of the level being left.
func (v *Game) takePets(actChange bool) {
	v.petCarry = nil

	if v.monsters == nil || v.localPlayer == nil {
		return
	}

	ox, oy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	c, followed, dropped := v.monsters.TakePetsForLevelChange(v.localPlayer, ox, oy, actChange)
	if followed+dropped > 0 {
		v.Infof("PETS level change actChange=%v: %d follow, %d released", actChange, followed, dropped)
	}

	if followed > 0 {
		v.petCarry = c
	}
}

// adoptPets puts the carried pets next to the hero in the new level.
func (v *Game) adoptPets() {
	c := v.petCarry
	v.petCarry = nil

	if c.Len() == 0 {
		return
	}

	d := v.monsterDirector()
	if d == nil {
		return
	}

	v.Infof("PETS arrive: %d of %d placed next to the hero", d.AdoptPets(c), c.Len())
}
