package d2quest

import "testing"

// The Hellforge handler of Cain's message 679 creates a soulstone without looking for one (0x5b3d00, wave 44).
func TestW44CainMessage679AlwaysGivesStone(t *testing.T) {
	g, _ := newGame(t)

	moveTo(g, 1, LevelPandemonium)
	talk(g, NPCTyrael2)
	kill(g, NPCDiablo, LevelChaosSanctum)
	pickup(g, ItemMephistoSoulstone)

	effects := g.Hear(NPCCain4, 679)
	if !effectCode(effects, EffectGiveItem, ItemMephistoSoulstone) {
		t.Fatalf("message 679 with a stone in the pack gave nothing: %+v", effects)
	}
}
