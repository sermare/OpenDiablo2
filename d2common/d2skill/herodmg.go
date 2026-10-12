package d2skill

// WeaponStats is implemented by a hero unit that knows its gear (wave 24). The numbers are the unit damage stats
// 0x15/0x16 (0x9f/0xa0 for a throwing weapon) before the percent pool, which MISSILE_BuildDamageDescriptor
// (0x64ca60, VERIFIED) builds apart from them. A unit without it gets WeaponDamage and no weapon percent.
type WeaponStats interface {
	// MissileWeaponDamage is the range a missile skill takes: the thrown range for a throwing weapon
	// (ITEM_IsThrowableType, stats 0x9f/0xa0), else the melee range.
	MissileWeaponDamage() (min, max int)
	// WeaponBonusPct is the equipped weapon's strength and dexterity bonus percent (0 unarmed).
	WeaponBonusPct() int
	// BareHandPct is the strength percent a bare handed melee blow gets (0x579120, VERIFIED), 0 when armed.
	BareHandPct() int
}

// missileWeapon is the weapon range of a missile skill's damage descriptor.
func missileWeapon(u Unit) (min, max int) {
	var wmin, wmax int

	if w, ok := u.(WeaponStats); ok {
		wmin, wmax = w.MissileWeaponDamage()
	} else {
		wmin, wmax = u.WeaponDamage()
	}

	if wmin < 1 {
		wmin = 1
	}

	if wmax < wmin+1 {
		wmax = wmin + 1
	}

	return wmin, wmax
}

// weaponBonusPct is the weapon's strength / dexterity percent of the descriptor's pool.
func weaponBonusPct(u Unit) int {
	if w, ok := u.(WeaponStats); ok {
		return w.WeaponBonusPct()
	}

	return 0
}

// meleeBonusPct is the percent a melee blow adds to the weapon range: the weapon's attribute bonus, or the
// strength of a bare handed blow.
func meleeBonusPct(u Unit) int {
	if w, ok := u.(WeaponStats); ok {
		return w.WeaponBonusPct() + w.BareHandPct()
	}

	return 0
}

// normalDamageFlat is the unit's item_normaldamage (stat 0x6f, the "+N to damage" property), which the weapon
// hit roll COMBAT_RollPhysicalDamage (0x579120, VERIFIED wave 46) adds to both ends of the range before the
// percent pool. It is read by that roll only: MISSILE_BuildDamageDescriptor (0x64ca60) and the thrown weapon
// description (0x4e7470) never read stat 0x6f, so missiles and thrown weapons ignore it (VERIFIED wave 46).
func normalDamageFlat(u Unit) int32 { return int32(u.Stat("item_normaldamage")) }
