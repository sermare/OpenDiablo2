package d2gamescreen

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2reward"
)

// The Hellforge reward shower (Game.exe 0x5b42d0, d2reward.Shower): after the
// forge is smashed with the hammer it drops a gem or skull per pulse from the
// table of the current stage (generator value modulo 7), the stage falls after
// every pulse that produced an item, pulses are 0x14 frames apart, and at stage
// 0 an expansion game gets one rune of the difficulty (roll of 11).
//
// VERIFIED (0x5b3820 operate, 0x5b42d0 tick, notes verify-diablo-hellforge.md): the smash sets the stage to 4 and
// the items per pulse (node data +0x14) to 1. The exe drops the items from the forge object itself; here they fall
// at the hero's feet (an engine placement, not the exe's).
const hellforgeCount = 1

// forgeShower is a running shower and its clock.
type forgeShower struct {
	s      *d2reward.Shower
	seed   *d2rand.Seed
	wait   float64 // frames until the next pulse
	pulses int
}

func newForgeShower(count int, expansion bool, difficulty int, seed uint32) *forgeShower {
	return &forgeShower{s: d2reward.NewShower(count, expansion, difficulty), seed: d2rand.New(seed)}
}

// advance runs the frames that passed; it returns the result of a pulse that
// happened (ok) and whether the shower is over.
func (f *forgeShower) advance(frames float64, drop func(code string) bool) (res d2reward.PulseResult, ok, over bool) {
	f.wait -= frames
	if f.wait > 0 {
		return res, false, false
	}

	res = f.s.Pulse(f.seed.Step, f.seed.Roll, drop)
	f.pulses++

	if res.Next > 0 {
		f.wait = float64(res.Next)

		return res, true, false
	}

	return res, true, true
}

// startHellforgeShower begins the shower of the smashed forge.
func (v *Game) startHellforgeShower() {
	if v.forge != nil {
		return
	}

	seed := uint32(0x5b42d0)
	if v.gameClient != nil && v.gameClient.MapEngine != nil {
		seed ^= uint32(v.gameClient.MapEngine.Seed())
	}

	v.forge = newForgeShower(hellforgeCount, v.heroExpansion(), v.questDifficulty(), seed)
	v.Infof("HELLFORGE shower starts stage=%d count=%d expansion=%v difficulty=%d", v.forge.s.Stage, hellforgeCount,
		v.forge.s.Expansion, v.forge.s.Difficulty)
}

// advanceHellforge runs the shower's clock.
func (v *Game) advanceHellforge(elapsed float64) {
	f := v.forge
	if f == nil {
		return
	}

	stage := f.s.Stage

	res, ok, over := f.advance(elapsed*questFrameRate, func(code string) bool {
		if v.asset.Records.Item.All[code] == nil {
			return false
		}

		v.spawnQuestItem(code)

		return true
	})
	if !ok {
		return
	}

	v.Infof("HELLFORGE pulse stage=%d dropped=[%s] next=%d", stage, strings.Join(res.Dropped, " "), res.Next)

	if res.Rune != "" {
		v.Infof("HELLFORGE rune %s dropped", res.Rune)
	}

	if over {
		v.Infof("HELLFORGE shower over after %d pulses", f.pulses)

		v.forge = nil
	}
}

// isHellforgeSmash reports whether a quest effect is the smashing of the forge
// (the third hammer blow on the opened forge destroys the hammer, VERIFIED 0x5b3820; the soulstone is consumed
// by the first operation, which only opens the forge).
func isHellforgeSmash(e d2quest.Effect) bool {
	return e.Kind == d2quest.EffectDeleteItem && e.Quest == d2quest.QuestHellforge && e.Code == d2quest.ItemHellforgeHammer
}
