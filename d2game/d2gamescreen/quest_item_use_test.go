package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

func TestQuestItemEffects(t *testing.T) {
	var rec d2s.QuestRecord

	g := d2quest.New(&rec, &d2s.NPCBlock{}, 0)

	// without the reward bits nothing happens, and unknown codes are not quest items
	for _, code := range []string{"ass", "xyz", "tr2"} {
		if eff, known := questItemEffects(g, code); !known || len(eff) != 0 {
			t.Errorf("%s without its quest bit: known=%v effects=%v", code, known, eff)
		}
	}

	if _, known := questItemEffects(g, "hp1"); known {
		t.Error("a health potion is not a quest item")
	}

	// Book of Skill: bit 5 of Radament's slot 9 gives one skill point once
	rec.Set(9, 5)

	eff, _ := questItemEffects(g, "ass ")
	if len(eff) != 1 || eff[0].Kind != d2quest.EffectSkillPoint || eff[0].Value != 1 || rec.Get(9, 5) {
		t.Fatalf("book: %v bit=%v", eff, rec.Get(9, 5))
	}

	if eff, _ := questItemEffects(g, "ass"); len(eff) != 0 {
		t.Error("a second book must do nothing")
	}

	// Potion of Life: bit 5 of slot 20, +20 once
	rec.Set(20, 5)

	eff, _ = questItemEffects(g, "xyz")
	if len(eff) != 1 || eff[0].Code != "life-boost" || eff[0].Value != 20 {
		t.Fatalf("potion: %v", eff)
	}

	if eff, _ := questItemEffects(g, "xyz"); len(eff) != 0 {
		t.Error("a second potion must do nothing")
	}

	// Scroll of Resistance: bit 8 given, bit 7 not yet read
	rec.Set(37, 8)

	if eff, _ := questItemEffects(g, "tr2"); len(eff) != 1 || eff[0].Code != "resist-bonus" {
		t.Fatalf("scroll: %v", eff)
	}

	if eff, _ := questItemEffects(g, "tr2"); len(eff) != 0 {
		t.Error("a second scroll read must do nothing")
	}
}
