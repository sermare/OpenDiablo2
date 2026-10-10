package d2monster

// FaithfulB lists the monai names from N to Z (case-insensitive) that were
// stand-ins (ai_rest.go) or unported and now have a port of the exe think
// function in the ai_fb_*.go files, for tests and tooling. Each name is
// registered by exactly one of those files.
func FaithfulB() []string {
	return []string{
		// ai_fb_1.go
		"ShadowMaster", "ShadowMasterNoInit", "ShadowWarrior", "Nihlathak", "WillOWisp", "Vampire", "SuccubusWitch",
		// ai_fb_2.go
		"ZakarumPriest", "ZakarumZealot", "OblivionKnight", "Overseer", "Regurgitator", "VileMother", "VileDog",
		"ThornHulk", "PinHead", "PutridDefiler", "QuillMother",
		// ai_fb_3.go
		"SiegeBeast", "SiegeTower", "ReanimatedHorde", "SuicideMinion", "Spirit", "TrappedSoul", "UberBaal",
		"Trap-Melee", "Trap-Missile", "Trap-RightArrow", "Trap-LeftArrow", "Trap-Poison", "Trap-Nova",
		"SandMaggotQueen", "Sarcophagus", "Tentacle", "TentacleHead", "FrogDemon", "NecroPet", "Vines", "Totem", "Raven",
		// ai_fb_npc.go
		"Navi", "Npc", "NpcBarb", "NpcOutOfTown", "NpcStationary", "Towner", "TownRogue", "Vendor", "Wussie",
	}
}
