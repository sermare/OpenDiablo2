package d2party

// Corpse loot permission (VERIFIED w45, Game.exe PLAYER_CanLootCorpse 0x57db10 / PLAYER_ServerLootCorpse 0x57db90):
// a player may loot a corpse when he is its owner, or when the owner's relation node toward him has flag 1 set
// (PLAYER_TestRelationFlags(owner, looter, 1)); anybody else is refused and the corpse is left alone. The packet
// table at 0x6e36c8 (0x5d/0x5e entries 1..4) sets one relation flag per entry: 2 and 4 by 0x5a32a0 / 0x5a3300,
// 8 (hostile) by 0x5a3870; that entry 1 is the loot flag the "allow loot" check box sends is UNVERIFIED (the code
// of entry 1 at 0x5a31d0 is not a defined function in Ghidra). The permission is per direction and is not
// reciprocal, a party does not grant it by itself.

// SetLootAllowed lets owner (true) or forbids (false) looter to loot owner's corpse.
func (r *Roster) SetLootAllowed(owner, looter string, allowed bool) error {
	o, ok := r.players[owner]
	if !ok || r.players[looter] == nil {
		return ErrUnknownPlayer
	}

	if owner == looter {
		return ErrSelf
	}

	if o.loot == nil {
		o.loot = map[string]bool{}
	}

	if allowed {
		o.loot[looter] = true
	} else {
		delete(o.loot, looter)
	}

	return nil
}

// CanLootCorpse reports whether looter may loot the corpse of owner.
func (r *Roster) CanLootCorpse(owner, looter string) bool {
	if owner == looter {
		return r.players[owner] != nil
	}

	o, ok := r.players[owner]

	return ok && r.players[looter] != nil && o.loot[looter]
}
