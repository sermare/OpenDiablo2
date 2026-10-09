package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio"
)

// speechAudio reports whether NPC speech is played (options NPC SPEECH is not
// TEXT ONLY); providers without the setting always play it.
func (v *Game) speechAudio() bool {
	if sp, ok := v.audioProvider.(interface{ SpeechAudio() bool }); ok {
		return sp.SpeechAudio()
	}

	return true
}

// playSpeech plays an NPC voice line by Sounds.txt index. Speech never
// overlaps: the line still playing is stopped first (its Fade Out applies).
// With NPC SPEECH on TEXT ONLY nothing is played.
func (v *Game) playSpeech(index int) *d2audio.Sound {
	if !v.speechAudio() {
		return nil
	}

	if v.speech != nil {
		v.speech.Stop()
		v.speech = nil
	}

	v.speech = v.soundEngine.PlaySoundID(index)

	return v.speech
}

// questStingers maps a killed boss (its name in the monster tables) to its
// Sounds.txt quest music. Which event the original plays each one on is not in
// the notes (UNVERIFIED); the pairing follows the sound file names
// (andarielaction.wav, bloodravenresolution.wav...).
var questStingers = map[string]string{
	"Andariel": "music_quest_andariel", "Blood Raven": "music_quest_bloodraven", "Radament": "music_quest_radament",
	"Duriel": "music_quest_horadric", "Mephisto": "music_quest_mephisto", "Izual": "music_quest_izual",
	"Diablo": "music_quest_diablo", "Baal": "music_quest_baal", "Nihlathak": "music_quest_nihlathak",
	"Shenk the Overseer": "music_quest_shenk", "The Summoner": "music_quest_tainted",
}

// denStinger is the quest music when the Den of Evil is cleared.
const denStinger = "music_quest_den"

// playStinger plays a quest or boss music sting over the area music.
func (v *Game) playStinger(handle, why string) {
	if handle == "" {
		return
	}

	if v.soundEngine.PlaySoundHandle(handle) != nil {
		v.Infof("STINGER %s (%s)", handle, why)
	}
}

// onItemSound plays the sound of an item moved by the hero in a panel.
func (v *Game) onItemSound(handle, kind string) {
	if v.localPlayer == nil {
		return
	}

	v.playSoundAtPos(handle, v.localPlayer.GetPosition(),
		d2audio.PlayOpts{Emitter: heroEmitter, Hero: true, Kind: kind, Who: v.localPlayer.Name()})
}
