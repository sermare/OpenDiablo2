package d2monster

// Batch 7 closes documented simplifications of earlier ground, melee, ranged
// and caster handlers by re-reading each think function in Game.exe 1.14b
// (notes: ai-faithful-batch7.md in the reverse-engineering notes). The
// handlers keep living in their original files; this list is what the
// progress bar counts.

// FaithfulBatch7 lists the handlers whose flow was re-verified against the exe
// and corrected in batch 7:
//
//	Fallen (alert command is never consumed by an attack), Brute (A1/A2 split is
//	aip4), Scarab (alert path, Jab without a roll, toggle roll, rally order),
//	SkeletonMage (shared distance variable), Fetish (target life percent),
//	FetishShaman (skill level gate, fetish corpse buddy, squared compare),
//	BatDemon (Aggressive term, flight stat, phase-2 switch), Megademon (skill
//	level gate and state 0xc), Succubus (target life, max life/mana, stat-list
//	flag), AbyssKnight (aurastate gate, no-target cast), Bighead (shots go to
//	the tick target) and CorruptRogue (run distance 20 - 3*scaling, run ends
//	the tick).
func FaithfulBatch7() []string {
	return []string{"Fallen", "Brute", "Scarab", "SkeletonMage", "Fetish", "FetishShaman", "BatDemon",
		"Megademon", "Succubus", "AbyssKnight", "Bighead", "CorruptRogue"}
}

// ReverifiedBatch7 lists handlers read against the exe in batch 7 that needed
// no flow change (not counted again): the ground group and the simple melee
// and ranged ones.
func ReverifiedBatch7() []string {
	return []string{"Skeleton", "Zombie", "Goatman", "Wraith", "Swarm", "QuillRat", "PantherWoman", "SandLeaper",
		"SandRaider", "Mummy", "SkeletonBow", "CorruptArcher", "FetishBlowgun", "DoomKnight", "Minion",
		"SuccubusWitch"}
}
