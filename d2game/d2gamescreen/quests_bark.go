package d2gamescreen

import (
	"math"
	"os"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

const (
	barkCheckSeconds = 0.5
	barkRangeTiles   = 7.0  // UNVERIFIED: the distance at which the rogue scout calls out
	barkCooldown     = 40.0 // seconds before the same NPC barks again (UNVERIFIED)
)

// advanceBarks makes Navi, the rogue scout (A1Q7), call out to a hero who comes
// near, as an overhead speech bubble with the voice line (message mode 3, the
// "bark" of the speech tables: no menu, no acknowledgement to the quest).
func (v *Game) advanceBarks(elapsed float64) {
	r := v.questRT

	if r.barkWait > 0 {
		r.barkWait -= elapsed
	}

	r.barkAcc += elapsed
	if r.barkAcc < barkCheckSeconds || r.barkWait > 0 {
		return
	}

	r.barkAcc = 0

	npc, ok := v.findNPC(d2quest.NPCNavi)
	if !ok {
		return
	}

	px, py := v.localPlayer.GetPositionF()
	nx, ny := npc.GetPositionF()

	if math.Hypot(px-nx, py-ny) > barkRangeTiles {
		return
	}

	lines := r.g.Activate(d2quest.NPCNavi).Lines
	if len(lines) == 0 {
		return
	}

	s := lines[0]
	snd, hasSound := d2quest.SoundForMessage(s.Msg)
	key := d2quest.TextKey(s.Msg)
	text := v.asset.TranslateString(key)

	if text == key {
		text = ""
	}

	text = speechText(text)

	r.barkWait = barkCooldown

	v.Infof("QUEST BARK npc=%q msg=%d mode=3 sound=%d handle=%s key=%s text=%q", npc.Label(), s.Msg, snd.Index, snd.Handle, key, shorten(text, 60))

	if hasSound && os.Getenv("OD2_AUTOTEST_MUTE") == "" {
		v.playSpeech(snd.Index)
	}

	if text != "" {
		v.gameControls.Speech.Show(npc.Label(), text, speechSeconds(text))
	}
}
