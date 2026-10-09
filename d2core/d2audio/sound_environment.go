package d2audio

import (
	"math"
	"math/rand"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2daynight"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio/d2sfx"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

// ticksPerSecond is the game tick rate Event Delay is expressed in (assumed
// to be the 25 Hz game frame; the column's unit is UNVERIFIED).
const ticksPerSecond = d2sfx.TicksPerSecond

// Selection is what a SoundEnviron.txt row plays at a given time of day.
type Selection struct {
	Song     int // music (Sounds.txt index, Music Vol master)
	Ambience int // looping scene sound
	Event    int // random one-shot (or short loop) sound
	Night    bool
}

// IsNight says whether a d2daynight phase uses the Night columns. The real
// game's night/day split for SoundEnviron.txt was not located in the binary
// (UNVERIFIED); this reuses the verified greeting classification: phase 1
// (morning) and phases 2 and 3 (day) are day, phases 0, 4 and 5 are night.
func IsNight(phase int) bool {
	return d2daynight.Classify(phase) == d2daynight.Evening
}

// Select picks the sounds of an environment for day or night.
func Select(r *d2records.SoundEnvironRecord, night bool) Selection {
	if r == nil {
		return Selection{Night: night}
	}

	if night {
		return Selection{Song: r.Song, Ambience: r.NightAmbience, Event: r.NightEvent, Night: true}
	}

	return Selection{Song: r.Song, Ambience: r.DayAmbience, Event: r.DayEvent}
}

// NextEventTicks returns the wait before the next ambient event for an Event
// Delay column value. The column is the nominal delay in game ticks; the real
// game's randomisation was not located, so the wait is uniform in
// [delay/2, 3*delay/2) (UNVERIFIED).
func NextEventTicks(delay int, rnd func(int) int) int {
	if delay <= 1 {
		return 1
	}

	return delay/2 + rnd(delay)
}

// EventPosition places an ambient event around the listener at a random
// bearing, between 15% and 90% of the sound's audible radius, in sound units.
// The bank doubles the y delta when measuring distance, so y is halved here to
// keep the distance on the circle.
func EventPosition(lx, ly, radius float64, rnd func(int) int) (x, y float64) {
	const steps = 1000

	angle := 2 * math.Pi * float64(rnd(steps)) / steps
	dist := radius * (0.15 + 0.75*float64(rnd(steps))/steps)

	return lx + dist*math.Cos(angle), ly + dist*math.Sin(angle)/2
}

// envSink is what a SoundEnvironment needs from the sound engine.
type envSink interface {
	// loop starts a non-positional looping sound; the returned function stops it.
	loop(index int) (stop func())
	// event plays a positional one-shot near the listener.
	event(index int, rnd func(int) int) (handle string)
	// handle names a Sounds.txt index for logs.
	handle(index int) string
}

type engineSink struct{ e *SoundEngine }

func (s engineSink) loop(index int) func() {
	snd := s.e.PlaySoundID(index)
	if snd == nil {
		return func() {}
	}

	return snd.Stop
}

func (s engineSink) handle(index int) string {
	if row := s.e.Bank().Table().Get(index); row != nil {
		return row.Handle
	}

	return ""
}

func (s engineSink) event(index int, rnd func(int) int) string {
	row := s.e.Bank().Table().Get(index)
	if row == nil {
		return ""
	}

	lx, ly := s.e.Listener()
	x, y := EventPosition(lx, ly, d2sfx.FalloffRadius(row.Falloff), rnd)

	s.e.play(d2sfx.Request{Index: index, HasPos: true, X: x, Y: y}, PlayOpts{X: x, Y: y, Kind: "ambient-event", Who: row.Handle})

	return row.Handle
}

// SoundEnvironment plays the music, ambient loop and random events of the
// map area the hero is in, from SoundEnviron.txt (selected by the level's
// SoundEnv column) and the day phase.
type SoundEnvironment struct {
	sink  envSink
	envs  map[int]*d2records.SoundEnvironRecord
	rnd   func(int) int
	logf  func(format string, args ...interface{})
	phase int

	environment *d2records.SoundEnvironRecord
	night       bool
	started     bool

	stopMusic   func()
	stopAmbient func()
	cur         Selection

	eventTimer float64 // game ticks until the next event
}

// NewSoundEnvironment creates a SoundEnvironment using the given SoundEngine
func NewSoundEnvironment(soundEngine *SoundEngine) SoundEnvironment {
	return newSoundEnvironment(engineSink{soundEngine}, soundEngine.asset.Records.Sound.Environment,
		// nolint:gosec // client-only, no need for a secure generator
		rand.Intn, soundEngine.Infof)
}

func newSoundEnvironment(sink envSink, envs map[int]*d2records.SoundEnvironRecord, rnd func(int) int,
	logf func(string, ...interface{})) SoundEnvironment {
	r := SoundEnvironment{
		sink: sink, envs: envs, rnd: rnd, logf: logf,
		phase: d2daynight.PhaseDay,
		// Start with env NONE
		environment: envs[0],
	}

	if r.environment == nil {
		r.environment = &d2records.SoundEnvironRecord{}
	}

	return r
}

// Environment returns the active SoundEnviron.txt row.
func (s *SoundEnvironment) Environment() *d2records.SoundEnvironRecord { return s.environment }

// Selection returns what is currently selected for playback.
func (s *SoundEnvironment) Selection() Selection { return s.cur }

// NextEventIn returns the game ticks left before the next ambient event.
func (s *SoundEnvironment) NextEventIn() float64 { return s.eventTimer }

// SetEnv sets the sound environment using the given record index
func (s *SoundEnvironment) SetEnv(environmentIdx int) {
	newEnv, ok := s.envs[environmentIdx]
	if !ok {
		return
	}

	if s.started && s.environment.Index == environmentIdx {
		return
	}

	s.environment = newEnv
	s.apply("env change")
}

// SetDayPhase tells the environment the d2daynight phase (0..5); a change
// between day and night swaps the ambience and the event sounds.
func (s *SoundEnvironment) SetDayPhase(phase int) {
	s.phase = phase

	night := IsNight(phase)
	if night == s.night {
		return
	}

	s.night = night
	if s.started {
		s.apply("day phase")
	}
}

// apply recomputes the selection, switching only the parts that changed (the
// old sound fades out through its Fade Out column while the new one fades in).
func (s *SoundEnvironment) apply(why string) {
	sel := Select(s.environment, s.night)
	first := !s.started
	s.started = true

	if first || sel.Song != s.cur.Song {
		s.stopMusic = s.restart(s.stopMusic, sel.Song)
	}

	if first || sel.Ambience != s.cur.Ambience {
		s.stopAmbient = s.restart(s.stopAmbient, sel.Ambience)
	}

	changedEvent := first || sel.Event != s.cur.Event || why == "env change"
	s.cur = sel

	if changedEvent {
		s.scheduleEvent()
	}

	s.logf("AMBIENT %s env=%s(%d) phase=%d night=%v music=%s(%d) ambience=%s(%d) event=%s(%d) event_delay=%d indoors=%d",
		why, s.environment.Handle, s.environment.Index, s.phase, s.night, s.sink.handle(sel.Song), sel.Song,
		s.sink.handle(sel.Ambience), sel.Ambience, s.sink.handle(sel.Event), sel.Event,
		s.environment.EventDelay, s.environment.Indoors)
}

func (s *SoundEnvironment) restart(stop func(), index int) func() {
	if stop != nil {
		stop()
	}

	if index == 0 {
		return nil
	}

	return s.sink.loop(index)
}

func (s *SoundEnvironment) scheduleEvent() {
	if s.cur.Event == 0 {
		s.eventTimer = math.Inf(1)
		return
	}

	s.eventTimer = float64(NextEventTicks(s.environment.EventDelay, s.rnd))
	s.logf("AMBIENT event scheduled handle=%s in=%.1fs (delay column %d ticks)",
		s.sink.handle(s.cur.Event), s.eventTimer/ticksPerSecond, s.environment.EventDelay)
}

// Advance advances the sound engine and plays sounds when necessary
func (s *SoundEnvironment) Advance(elapsed float64) {
	if !s.started || s.cur.Event == 0 {
		return
	}

	s.eventTimer -= elapsed * ticksPerSecond
	if s.eventTimer > 0 {
		return
	}

	s.sink.event(s.cur.Event, s.rnd)
	s.scheduleEvent()
}
