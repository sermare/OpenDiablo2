package d2inventory

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// The timed-state potions: pSpell 6 (antidote, thawing) first ends the states in cstate1/cstate2 and then
// applies its own state (ITEM_RemoveOldStatesThenApplyTimedState 0x5bcb80); pSpell 9 (stamina) only applies it
// (ITEM_ApplyTimedStateFromItemRecord 0x5bc990). Both VERIFIED in the exe. The state lasts len frames
// (25 per second), its stats are stat1..3 with calc1..3. A stamina recovery stat (0x1c) above zero also tops
// the stamina up to the maximum (the code writes max + 0x100 when stamina is below max; the 8.8 shift of the
// two reads was not checked, so the exact value is U, the refill to full is the reading used here).

const (
	spellStateCure  = 6
	spellStateApply = 9

	statStaminaRecovery = "staminarecoverybonus"
)

// StateMod is one stat of a potion state (ItemStatCost name).
type StateMod struct {
	Stat  string
	Value int
}

// StatePotion is what a pSpell 6 or 9 potion does.
type StatePotion struct {
	State         string
	Cure          []string
	Frames        int
	Seconds       float64
	Mods          []StateMod
	RefillStamina bool
}

// StatePotionOf reads the timed-state effect of a misc.txt potion; ok is false for every other kind.
func StatePotionOf(rec *d2records.ItemCommonRecord) (p StatePotion, ok bool) {
	if rec == nil || !rec.Useable || (rec.SpellType != spellStateCure && rec.SpellType != spellStateApply) {
		return p, false
	}

	p.State = rec.OverlayState
	p.Frames = rec.EffectLength
	p.Seconds = float64(rec.EffectLength) / potionFramesPerSecond

	if rec.SpellType == spellStateCure {
		for _, c := range rec.CureOverlayStates {
			if c != "" {
				p.Cure = append(p.Cure, c)
			}
		}
	}

	for _, u := range rec.UsageStats {
		if u.Stat == "" {
			continue
		}

		v := calcNumber(string(u.Calc))
		if v == 0 {
			continue
		}

		p.Mods = append(p.Mods, StateMod{Stat: u.Stat, Value: v})

		if strings.EqualFold(u.Stat, statStaminaRecovery) && v > 0 {
			p.RefillStamina = true
		}
	}

	return p, p.State != "" && p.Frames > 0
}
