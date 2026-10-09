package d2gamescreen

import (
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
)

// OD2_AUTOEQUIP=1 runs the equip rule scenario (d2player.GameControls.AutoEquip)
// a few seconds after the hero appears and logs "EQUIP auto" lines. With
// OD2_AUTOMONSTER set the game then goes on fighting, and the armor and weapon
// durability changes show up as "DURABILITY" lines (OD2_DURABILITY_CHANCE=<percent>
// raises the loss chance so a short fight shows them). Without a monster scenario
// the game exits at the end (OD2_AUTOEXIT).
const autoEquipDelay = 3.0

type autoEquipState struct {
	elapsed float64
	done    bool
}

// playHeroUISound plays a Sounds.txt handle (a voice line or a UI sound) as the hero.
func (v *Game) playHeroUISound(handle string) {
	if v.localPlayer == nil {
		return
	}

	v.Infof("SOUND hero voice handle=%s", handle)
	v.playSoundAtPos(handle, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: "hero-voice", Who: v.localPlayer.Name()})
}

func (v *Game) advanceAutoEquip(elapsed float64) {
	a := &v.autoEquip
	if os.Getenv("OD2_AUTOEQUIP") == "" || a.done || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	if a.elapsed += elapsed; a.elapsed < autoEquipDelay {
		return
	}

	a.done = true

	v.gameControls.AutoEquip()

	if os.Getenv("OD2_AUTOMONSTER") == "" {
		v.autoTestExit()
	}
}
