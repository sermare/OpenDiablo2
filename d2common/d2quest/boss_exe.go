package d2quest

// Kill bits of the end bosses as the binary sets them (Game.ExeBossBits).
//
// VERIFIED (Game.exe 1.14b, read-only Ghidra):
//   - Mephisto, QUEST_A3_TheGuardian_CompleteQuest 0x5b9d00: slot 22 bits 0xd
//     (primary goal), 0 (done) and 0xb; applies to the killer when slot 22
//     bit 0 is clear (0x5ba440) or when a hero in level 102 sees the quest
//     still open (0x5b9d50, bit 0xb clear).
//     The same handler stamps the item code "mss " (Mephisto's Soulstone) on
//     the dying unit and drops it through 0x557980; Mephistoq in
//     treasureclassex has no soulstone, so the drop is the quest's.
//   - Diablo, QUEST_A4_TerrorsEnd_CompleteQuest 0x5b2950: slot 26 bits 0xd and
//     0, then the progress bits are cleared (QUESTREC_ClearProgressBits, which
//     bits is not read), and in the classic game (game+0x70 == 0) bits 6 and 7.
//     Applied to heroes in Diablo's room or a neighbour (0x5b2d70) whose bits 0
//     and 1 are clear.
//   - Baal, 0x58bae0 (from the callback 0x58bc60 of 0x58bce0): slot 40 bits 0xd
//     and 0 for heroes in level 132 (the Worldstone Chamber) whose bits 0 and 1
//     are clear.
//
// None of the three sets bit 1 (reward pending): there is no reward talk, the
// quest is done at once. The engine's own flow marks them reward pending and
// pays the claim effects when the hero speaks; ExeBossBits keeps those
// effects but grants at once.
const (
	exeBitMephisto = 11 // 0xb, set by the Guardian's completion only

	// LevelWorldstoneChamber is the level of Baal's death (levels.txt 132).
	LevelWorldstoneChamber = 132
)

// exeKill describes the exe's extra rules of one boss kill trigger.
type exeKill struct {
	bits        []int  // extra bits set with done + primary goal
	classicBits []int  // extra bits in the classic game (Expansion false)
	level       int    // the hero must be in this level (0: any)
	dropCode    string // quest item dropped at the corpse (unit item-code override +0xb8)
}

func (x *exeKill) applies(g *Game, e *Event) bool {
	if x.level == 0 {
		return true
	}

	lvl := e.Level
	if lvl == 0 {
		lvl = g.Level
	}

	return lvl == x.level
}

// finish sets the extra bits and pays the claim effects of the quest, which
// the engine would otherwise hand out when the reward line is spoken.
func (x *exeKill) finish(g *Game, q *Quest, s *spec) {
	for _, b := range x.bits {
		g.set(q, b, "exe boss kill bit")
	}

	if !g.Expansion {
		for _, b := range x.classicBits {
			g.set(q, b, "exe boss kill bit (classic)")
		}
	}

	if x.dropCode != "" {
		g.emit(Effect{Kind: EffectGiveItem, Quest: q.ID, Code: x.dropCode,
			Note: "quest drop at the corpse (exe: unit code override, 0x557980)"})
	}

	if s.claimFx != nil {
		for _, f := range s.claimFx(g, q) {
			g.emit(f)
		}
	}
}
