package ebiten

import (
	"io"
	"math"
	"sync"
	"sync/atomic"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

type panStream struct {
	io.ReadSeeker
	pan  float64 // -1: left; 0: center; 1: right
	Lock sync.Mutex

	// what the audio device pulled from this stream (audit, see SoundEffect.Flow)
	read int64
	peak int16
}

const (
	bitsPerByte = 8
)

func newPanStreamFromReader(src io.ReadSeeker) *panStream {
	return &panStream{
		ReadSeeker: src,
		pan:        0,
	}
}

func (s *panStream) Read(p []byte) (n int, err error) {
	s.Lock.Lock()
	defer s.Lock.Unlock()

	n, err = s.ReadSeeker.Read(p)
	if err != nil && n == 0 {
		return
	}

	defer func() { s.note(p[:n]) }()

	if err != nil {
		return
	}

	ls := math.Min(s.pan*-1+1, 1)
	rs := math.Min(s.pan+1, 1)

	for i := 0; i < len(p); i += 4 {
		lc := int16(float64(int16(p[i])|int16(p[i+1])<<bitsPerByte) * ls)
		rc := int16(float64(int16(p[i+2])|int16(p[i+3])<<bitsPerByte) * rs)

		p[i] = byte(lc)
		p[i+1] = byte(lc >> bitsPerByte)
		p[i+2] = byte(rc)
		p[i+3] = byte(rc >> bitsPerByte)
	}

	return
}

// note records the amount and the loudest 16-bit sample of data handed to the device.
func (s *panStream) note(p []byte) {
	s.read += int64(len(p))

	for i := 0; i+1 < len(p); i += 2 {
		v := int16(p[i]) | int16(p[i+1])<<bitsPerByte
		if v < 0 {
			v = -v
		}

		if v > s.peak {
			s.peak = v
		}
	}
}

// Flow reports how many bytes the audio device has pulled from this effect and
// the loudest sample among them (0..32767), after panning. Used by the
// in-game audio audit: a started sound that never flows is silent.
func (v *SoundEffect) Flow() (bytes int64, peak int) {
	v.panStream.Lock.Lock()
	defer v.panStream.Lock.Unlock()

	return v.panStream.read, int(v.panStream.peak)
}

// totalPlays counts every sound effect started through any path (audit).
var totalPlays int64

// SoundEffect represents an ebiten implementation of a sound effect
type SoundEffect struct {
	player      *audio.Player
	volumeScale float64
	panStream   *panStream
}

// SetPan sets the audio pan, left is -1.0, center is 0.0, right is 1.0
func (v *SoundEffect) SetPan(pan float64) {
	v.panStream.Lock.Lock()
	v.panStream.pan = pan
	v.panStream.Lock.Unlock()
}

// SetVolumeScale replaces the provider's master volume factor of this effect
// (the sound engine applies the master volumes itself and sets 1).
func (v *SoundEffect) SetVolumeScale(scale float64) {
	v.volumeScale = scale
}

// SetVolume ets the volume
func (v *SoundEffect) SetVolume(volume float64) {
	v.player.SetVolume(volume * v.volumeScale)
}

// IsPlaying returns a bool for whether or not the sound is currently playing
func (v *SoundEffect) IsPlaying() bool {
	return v.player.IsPlaying()
}

// Play plays the sound effect
func (v *SoundEffect) Play() {
	atomic.AddInt64(&totalPlays, 1)

	err := v.player.Rewind()

	if err != nil {
		panic(err)
	}

	v.player.Play()
}

// Stop stops the sound effect
func (v *SoundEffect) Stop() {
	v.player.Pause()
}
