package d2gamescreen

import (
	"os"

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
	return v.playSpeechKind(index, "npc-speech")
}

// speechPlays says whether voice lines are started. They are not under
// OD2_AUTOTEST_MUTE, except when OD2_SOUNDLOG records the sounds: the engine
// then runs them on its silent players so the log can assert them.
func speechPlays() bool {
	return os.Getenv("OD2_AUTOTEST_MUTE") == "" || os.Getenv("OD2_SOUNDLOG") != ""
}

// playSpeechKind is playSpeech with the SOUNDLOG kind (npc-speech, npc-greeting).
func (v *Game) playSpeechKind(index int, kind string) *d2audio.Sound {
	if !v.speechAudio() {
		return nil
	}

	if v.speech != nil {
		v.speech.Stop()
		v.speech = nil
	}

	v.speech = v.soundEngine.PlaySoundKind(index, kind)

	return v.speech
}

// questKillVoices maps a boss killed by the hero to the hero voice id the server sends (packet 0x2c
// sound ids, UNITSND_PlayUnitSoundById 0x4c8470): the hero's completion line together with the quest
// sting and the silence of the level music (d2audio.QuestPlans; the ids and what each plays are VERIFIED
// in wave 37, that the kill is the event that sends them is UNVERIFIED but matches the line names).
var questKillVoices = map[string]int{
	"Andariel": 0x21, "Blood Raven": 0x22, "Radament": 0x32, "Mephisto": 0x42, "Diablo": 0x4b,
	"Shenk the Overseer": 0x50, "Nihlathak": 0x52, "Baal": 0x53,
}

// questStingers maps the other killed bosses to their Sounds.txt quest music. Which event the original
// plays these on is not found in the exe (UNVERIFIED); the pairing follows the sound file names
// (horadricaction.wav, izualaction.wav). The Summoner is not here: its kill is the hero line 0x33 and
// the Tainted Sun sting belongs to the Amulet of the Viper (0x34, sent by the quest effects).
var questStingers = map[string]string{
	"Duriel": "music_quest_horadric", "Izual": "music_quest_izual",
}

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

// interruptSpeech cuts off the NPC voice and the subtitle that is on screen: the player clicked another NPC or
// opened a vendor window, and the original stops the running line then (a subtitle, which is this fork's own, must
// not stay over the trade window; wave 41 hunt: Warriv's 25 s subtitle covered Akara's and Charsi's shop).
func (v *Game) interruptSpeech() {
	if v.speech != nil {
		v.speech.Stop()
		v.speech = nil
	}

	if v.gameControls != nil && v.gameControls.Speech != nil {
		v.gameControls.Speech.Clear()
	}
}

// skipSpeechByKey is Esc / Space during an NPC line (exe handler 0x49de80 -> 0x49d090, VERIFIED): the voice and
// the subtitle are cut and the talk ends.
func (v *Game) skipSpeechByKey() {
	v.interruptSpeech()
	v.Infof("NPC speech skipped by key")
}
