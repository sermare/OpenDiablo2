package d2inventory

import (
	"math"
	"testing"
)

// The rows of misc.txt 1.14b: antidote (pSpell 6, state antidote, cstate1 poison, len 750), thawing (pSpell 6,
// cstate1 freeze, cstate2 cold) and stamina (pSpell 9, staminarecoverybonus 5000).
func TestStatePotionOf(t *testing.T) {
	anti := potion(6, 750, "poisonresist", "50", "maxpoisonresist", "10")
	anti.OverlayState = "antidote"
	anti.CureOverlayStates = [2]string{"poison", ""}

	p, ok := StatePotionOf(anti)
	if !ok || p.State != "antidote" || p.Frames != 750 || math.Abs(p.Seconds-30) > 1e-9 {
		t.Fatalf("antidote = %+v ok=%v", p, ok)
	}

	if len(p.Cure) != 1 || p.Cure[0] != "poison" || len(p.Mods) != 2 || p.Mods[0] != (StateMod{"poisonresist", 50}) ||
		p.Mods[1] != (StateMod{"maxpoisonresist", 10}) || p.RefillStamina {
		t.Errorf("antidote details = %+v", p)
	}

	thaw := potion(6, 750, "coldresist", "50", "maxcoldresist", "10")
	thaw.OverlayState = "thawing"
	thaw.CureOverlayStates = [2]string{"freeze", "cold"}

	if p, ok = StatePotionOf(thaw); !ok || len(p.Cure) != 2 || p.Cure[1] != "cold" {
		t.Errorf("thawing = %+v ok=%v", p, ok)
	}

	stam := potion(9, 750, "staminarecoverybonus", "5000")
	stam.OverlayState = "staminapot"
	stam.CureOverlayStates = [2]string{"poison", ""} // pSpell 9 never cures (0x5bc990 does not touch cstate)

	if p, ok = StatePotionOf(stam); !ok || len(p.Cure) != 0 || !p.RefillStamina || p.State != "staminapot" {
		t.Errorf("stamina = %+v ok=%v", p, ok)
	}

	heal := potion(3, 256, "hpregen", "320")
	heal.OverlayState = "healthpot"

	if _, ok = StatePotionOf(heal); ok {
		t.Error("a healing potion is not a state potion")
	}

	if _, ok = StatePotionOf(nil); ok {
		t.Error("nil record")
	}

	noLen := potion(9, 0, "staminarecoverybonus", "5000")
	noLen.OverlayState = "staminapot"

	if _, ok = StatePotionOf(noLen); ok {
		t.Error("a state with no length is not applied (0x5bc990 returns 0 for len < 1)")
	}
}
