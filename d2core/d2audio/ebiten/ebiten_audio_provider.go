// Package ebiten contains ebiten's implementation of the audio interface
package ebiten

import (
	"fmt"
	"io"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio/d2soundpath"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

const sampleRate = 44100

const logPrefix = "Ebiten Audio Provider"

var _ d2interface.AudioProvider = &AudioProvider{} // Static check to confirm struct conforms to interface

// CreateAudio creates an instance of ebiten's audio provider
func CreateAudio(l d2util.LogLevel, am *d2asset.AssetManager) *AudioProvider {
	result := &AudioProvider{
		asset: am,
	}

	result.Logger = d2util.NewLogger()
	result.Logger.SetLevel(l)
	result.Logger.SetPrefix(logPrefix)

	result.audioContext = audio.NewContext(sampleRate)

	return result
}

// AudioProvider represents a provider capable of playing audio
type AudioProvider struct {
	asset        *d2asset.AssetManager
	audioContext *audio.Context // The Audio context
	bgmAudio     *audio.Player  // The audio player
	bgmStream    *wav.Stream
	lastBgm      string
	sfxVolume    float64
	bgmVolume    float64
	textOnly     bool // options: NPC speech set to TEXT ONLY

	*d2util.Logger
}

// PlayBGM loads an audio stream and plays it in the background
func (eap *AudioProvider) PlayBGM(song string) {
	if eap.lastBgm == song {
		return
	}

	eap.lastBgm = song

	if song == "" && eap.bgmAudio != nil && eap.bgmAudio.IsPlaying() {
		eap.bgmAudio.Pause()
		return
	}

	if eap.bgmAudio != nil {
		err := eap.bgmAudio.Close()

		if err != nil {
			eap.Fatal(err.Error())
		}
	}

	audioStream, err := eap.asset.LoadFileStream(song)

	if err != nil {
		eap.Warning("skipping missing music: " + err.Error())
		return
	}

	if _, err = audioStream.Seek(0, io.SeekStart); err != nil {
		eap.Fatal(err.Error())
	}

	eap.bgmStream, err = wav.Decode(eap.audioContext, audioStream)

	if err != nil {
		eap.Fatal(err.Error())
	}

	s := audio.NewInfiniteLoop(eap.bgmStream, eap.bgmStream.Length())
	eap.bgmAudio, err = audio.NewPlayer(eap.audioContext, s)

	if err != nil {
		eap.Fatal(err.Error())
	}

	eap.bgmAudio.SetVolume(eap.bgmVolume)

	// Play the infinite-length stream. This never ends.
	err = eap.bgmAudio.Rewind()

	if err != nil {
		panic(err)
	}

	eap.bgmAudio.Play()
}

// LoadSound loads a sound affect so that it canb e played
func (eap *AudioProvider) LoadSound(sfx string, loop, bgm bool) (d2interface.SoundEffect, error) {
	volume := eap.sfxVolume
	if bgm {
		volume = eap.bgmVolume
	}

	result, err := eap.createSoundEffect(sfx, eap.audioContext, loop)
	if err != nil {
		return nil, err
	}

	result.volumeScale = volume
	result.SetVolume(volume)

	return result, nil
}

// SetSpeechAudio says whether NPC speech is played (options NPC SPEECH is not TEXT ONLY).
func (eap *AudioProvider) SetSpeechAudio(on bool) { eap.textOnly = !on }

// SpeechAudio reports whether NPC speech should be played.
func (eap *AudioProvider) SpeechAudio() bool { return !eap.textOnly }

// Volumes returns the music and sound master volumes (0..1).
func (eap *AudioProvider) Volumes() (bgm, sfx float64) {
	return eap.bgmVolume, eap.sfxVolume
}

// SetVolumes sets the volumes of the audio provider
func (eap *AudioProvider) SetVolumes(bgmVolume, sfxVolume float64) {
	eap.sfxVolume = sfxVolume
	eap.bgmVolume = bgmVolume
}

// createSoundEffect creates a new instance of ebiten's sound effect implementation.
func (eap *AudioProvider) createSoundEffect(sfx string, context *audio.Context,
	loop bool) (*SoundEffect, error) {
	result := &SoundEffect{}

	name := sfx
	if entry, exists := eap.asset.Records.Sound.Details[sfx]; exists {
		name = entry.FileName
	}

	// Sounds.txt names are relative to the sfx, speech or music archive folder, with backslashes
	soundFile, found := d2soundpath.Resolve(name, func(p string) bool {
		ok, _ := eap.asset.FileExists(p)
		return ok
	})
	if !found {
		return nil, fmt.Errorf("sound %q not found in the sfx, speech or music folders", name)
	}

	audioData, err := eap.asset.LoadFileStream(soundFile)
	if err != nil {
		return nil, err
	}

	d, err := wav.Decode(context, audioData)

	if err != nil {
		eap.Fatal(err.Error())
	}

	if d == nil {
		eap.Fatal("Decoded data is nil")
	}

	var player *audio.Player

	if loop {
		s := audio.NewInfiniteLoop(d, d.Length())
		result.panStream = newPanStreamFromReader(s)
		player, err = audio.NewPlayer(context, result.panStream)
	} else {
		result.panStream = newPanStreamFromReader(d)
		player, err = audio.NewPlayer(context, result.panStream)
	}

	if err != nil {
		eap.Fatal(err.Error())
	}

	result.player = player

	return result, nil
}
