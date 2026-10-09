package d2audio

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2asset"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2audio/d2sfx"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

const (
	logPrefix = "Sound Engine"
)

// Sound is a handle on one sound started through the SoundEngine.
type Sound struct {
	bank   *d2sfx.Bank
	inst   *d2sfx.Instance
	handle string
}

// SetPan sets the stereo pan, range -1 to 1 (ignored for positional sounds,
// whose pan follows the emitter).
func (s *Sound) SetPan(pan float64) {
	s.bank.SetPan(s.inst, pan)
}

// Play is kept for API compatibility; the engine starts sounds itself.
func (s *Sound) Play() {}

// Stop the sound, only required for looping sounds. Honors the Fade Out column.
func (s *Sound) Stop() {
	s.bank.Stop(s.inst)
}

// Report returns the engine's latest decision about this sound.
func (s *Sound) Report() d2sfx.Report {
	return s.inst.Report()
}

// Mix returns the volume (0..1, after distance falloff) and pan (-1..1) the
// engine computes for this sound right now.
func (s *Sound) Mix() (vol, pan float64) {
	return s.bank.Mix(s.inst)
}

// String returns the sound handle
func (s *Sound) String() string {
	return s.handle
}

// SoundEngine provides functions for playing sounds. Voice limiting, priority,
// Group Size variants, Defer/Stop Inst, fades and distance falloff follow the
// original game's rules, implemented in package d2sfx.
type SoundEngine struct {
	asset    *d2asset.AssetManager
	provider d2interface.AudioProvider
	bank     *d2sfx.Bank
	ticks    float64
	mute     bool
	trace    bool
	stats    TraceStats
	lx, ly   float64 // listener in sound units, see SubtileToSound
	sounds   map[*Sound]struct{}

	*d2util.Logger
}

// clock converts engine time to game ticks (25/s).
type engineClock struct{ e *SoundEngine }

func (c engineClock) Now() int64 { return int64(c.e.ticks) }

// NewSoundEngine creates a new sound engine
func NewSoundEngine(provider d2interface.AudioProvider,
	asset *d2asset.AssetManager, l d2util.LogLevel, term d2interface.Terminal) *SoundEngine {
	r := SoundEngine{
		asset:    asset,
		provider: provider,
		sounds:   map[*Sound]struct{}{},
		mute:     os.Getenv("OD2_AUTOTEST_MUTE") != "",
	}

	r.Logger = d2util.NewLogger()
	r.Logger.SetPrefix(logPrefix)
	r.Logger.SetLevel(l)

	if err := term.Bind("playsoundid", "plays the sound for a given id", []string{"id"}, r.commandPlaySoundID); err != nil {
		r.Error(err.Error())
		return nil
	}

	if err := term.Bind("playsound", "plays the sound for a given handle string", []string{"name"}, r.commandPlaySound); err != nil {
		r.Error(err.Error())
		return nil
	}

	if err := term.Bind("activesounds", "list currently active sounds", nil, r.commandActiveSounds); err != nil {
		r.Error(err.Error())
		return nil
	}

	if err := term.Bind("killsounds", "kill active sounds", nil, r.commandKillSounds); err != nil {
		r.Error(err.Error())
		return nil
	}

	return &r
}

// rowsFromRecords converts Sounds.txt records into d2sfx rows.
func rowsFromRecords(details d2records.SoundDetails) []d2sfx.Row {
	rows := make([]d2sfx.Row, 0, len(details))

	for _, e := range details {
		rows = append(rows, d2sfx.Row{
			Handle: e.Handle, Index: e.Index, FileName: e.FileName, Volume: e.Volume,
			GroupSize: e.GroupSize, Loop: e.Loop, FadeIn: e.FadeIn, FadeOut: e.FadeOut,
			DeferInst: e.DeferInst, StopInst: e.StopInst, Duration: e.Duration,
			Compound: e.Compound, Falloff: e.Falloff, Priority: e.Priority,
			AsyncOnly: e.AsyncOnly, Stream: e.Stream, Stereo: e.Stereo, Tracking: e.Tracking,
			Solo: e.Solo, MusicVol: e.MusicVol,
		})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Index < rows[j].Index })

	return rows
}

// Bank returns the voice bank, building it from Sounds.txt on first use.
func (s *SoundEngine) Bank() *d2sfx.Bank {
	if s.bank != nil {
		return s.bank
	}

	table := d2sfx.NewTable(rowsFromRecords(s.asset.Records.Sound.Details))
	// nolint:gosec // client-only, no need for a secure generator
	s.bank = d2sfx.NewBank(table, d2sfx.DefaultVoices, s.loadPlayer, engineClock{s}, rand.Intn)

	return s.bank
}

func (s *SoundEngine) loadPlayer(row *d2sfx.Row) (d2sfx.Player, error) {
	if s.mute {
		return &mutePlayer{e: s, loop: row.Loop}, nil
	}

	return s.provider.LoadSound(row.FileName, row.Loop, row.MusicVol)
}

// mutePlayer stands in for audio under OD2_AUTOTEST_MUTE: it occupies a voice
// like a real player (one-shots end after one second) but makes no sound.
type mutePlayer struct {
	e       *SoundEngine
	loop    bool
	started float64
	on      bool
}

func (m *mutePlayer) Play()             { m.on, m.started = true, m.e.ticks }
func (m *mutePlayer) Stop()             { m.on = false }
func (m *mutePlayer) SetPan(float64)    {}
func (m *mutePlayer) SetVolume(float64) {}
func (m *mutePlayer) IsPlaying() bool {
	return m.on && (m.loop || m.e.ticks-m.started < d2sfx.TicksPerSecond)
}

// SetListener tells the engine where the hero is, for positional sounds, in
// sound units (see SubtileToSound).
func (s *SoundEngine) SetListener(x, y float64) {
	s.lx, s.ly = x, y
	s.Bank().SetListener(x, y)
}

// SetListenerSubtile is SetListener for a position in map subtiles.
func (s *SoundEngine) SetListenerSubtile(x, y float64) {
	s.SetListener(SubtileToSound(x, y))
}

// Listener returns the listener position in sound units.
func (s *SoundEngine) Listener() (x, y float64) { return s.lx, s.ly }

// SetTrace makes every positional sound log a SOUNDAT line with the distance,
// gain, volume and pan the engine chose (used by the OD2_AUTO* scenarios).
func (s *SoundEngine) SetTrace(on bool) { s.trace = on }

// TraceStats counts the positional sounds seen while tracing.
type TraceStats struct {
	Total, Audible, Inaudible int
	// Nearest and Farthest are emitter distances in sound units.
	Nearest, Farthest float64
	// ByKind counts sounds per PlayOpts.Kind.
	ByKind map[string]int
}

// TraceStats returns the counters of the positional sounds played since
// tracing was enabled.
func (s *SoundEngine) TraceStats() TraceStats { return s.stats }

// Subtile geometry in sound units. The bank measures distance as the hero
// relative delta with y doubled (verified) and the falloff radii (400..2000)
// look like screen pixels (inferred), so emitters are placed at their
// isometric screen position: one subtile is 16 px across and 8 px down (a
// 80x40 px tile holds 5x5 subtiles). Doubling y then gives an isotropic circle.
const (
	subtilePixelsX = 16.0
	subtilePixelsY = 8.0
)

// SubtileToSound converts a map position in subtiles to sound units.
func SubtileToSound(x, y float64) (sx, sy float64) {
	return (x - y) * subtilePixelsX, (x + y) * subtilePixelsY
}

// Advance updates sound engine state, triggering envelopes and cleanup
func (s *SoundEngine) Advance(elapsed float64) {
	s.ticks += elapsed * d2sfx.TicksPerSecond

	if s.bank == nil {
		return
	}

	s.bank.Advance()

	for sound := range s.sounds {
		if !sound.inst.Alive() {
			delete(s.sounds, sound)
		}
	}
}

// UnbindTerminalCommands unbinds commands from the terminal
func (s *SoundEngine) UnbindTerminalCommands(term d2interface.Terminal) error {
	return term.Unbind("playsoundid", "playsound", "activesounds", "killsounds")
}

// Reset stop all sounds and reset state
func (s *SoundEngine) Reset() {
	if s.bank != nil {
		s.bank.StopAll()
	}

	for snd := range s.sounds {
		delete(s.sounds, snd)
	}
}

// PlaySoundID plays a sound by Sounds.txt index (the row ordinal). It returns
// nil when the engine decided not to play it (index 0, no file, no voice...).
func (s *SoundEngine) PlaySoundID(id int) *Sound {
	return s.play(d2sfx.Request{Index: id})
}

// PlaySoundAt plays a positional sound; x and y are in the same units as
// SetListener. emitter is an opaque id used by Defer/Stop Inst (0 = none).
func (s *SoundEngine) PlaySoundAt(id int, x, y float64, emitter int) *Sound {
	return s.play(d2sfx.Request{Index: id, HasPos: true, X: x, Y: y, Emitter: emitter},
		PlayOpts{X: x, Y: y, Emitter: emitter, Kind: "at"})
}

// PlayOpts describes a positional play request.
type PlayOpts struct {
	// X, Y is the emitter in sound units; use SubtileToSound for map positions.
	X, Y float64
	// Emitter identifies the source for Defer/Stop Inst (0 = none).
	Emitter int
	// Delay is the number of game ticks (25/s) before the sound may start.
	Delay int
	// Volume overrides the Sounds.txt Volume column when > 0.
	Volume int
	// Hero marks the local hero as the emitter (priority bonus, verified).
	Hero bool
	// Who and Kind only label the SOUNDAT trace line.
	Who, Kind string
}

// PlayHandleAt plays a Sounds.txt handle at a position. Unknown handles and
// empty ones return nil without a warning (tables name sounds that the
// install may lack).
func (s *SoundEngine) PlayHandleAt(handle string, o PlayOpts) *Sound {
	e, ok := s.asset.Records.Sound.Details[handle]
	if !ok || handle == "" {
		return nil
	}

	return s.play(d2sfx.Request{Index: e.Index, HasPos: true, X: o.X, Y: o.Y, Emitter: o.Emitter,
		Delay: int64(o.Delay), Volume: o.Volume, Hero: o.Hero}, o)
}

func (s *SoundEngine) play(req d2sfx.Request, opts ...PlayOpts) *Sound {
	if req.Index == 0 {
		return nil
	}

	inst := s.Bank().Play(req)
	r := inst.Report()
	s.Debugf("sound %s", r)

	if s.trace && req.HasPos {
		var o PlayOpts
		if len(opts) > 0 {
			o = opts[0]
		}

		vol, pan := s.bank.Mix(inst)
		// the report's distance is only filled once a sound starts; recompute (y doubled, verified)
		dx, dy := req.X-s.lx, 2*(req.Y-s.ly)
		dist := math.Hypot(dx, dy)

		s.stats.Total++

		if s.stats.ByKind == nil {
			s.stats.ByKind = map[string]int{}
		}

		s.stats.ByKind[o.Kind]++

		if r.Decision == d2sfx.DecisionInaudible {
			s.stats.Inaudible++
		} else {
			s.stats.Audible++
		}

		if s.stats.Total == 1 || dist < s.stats.Nearest {
			s.stats.Nearest = dist
		}

		if dist > s.stats.Farthest {
			s.stats.Farthest = dist
		}

		if row := inst.Row(); row != nil {
			s.Infof("SOUNDAT kind=%s who=%q handle=%s file=%q pos=(%.0f,%.0f) dist=%.0f radius=%.0f gain=%.2f vol=%.2f pan=%+.2f delay=%d decision=%s",
				o.Kind, o.Who, r.Handle, r.File, req.X, req.Y, dist, d2sfx.FalloffRadius(row.Falloff),
				d2sfx.DistanceGain(dist*dist, row.Falloff), vol, pan, req.Delay, r.Decision)
		}
	}

	switch r.Decision {
	case d2sfx.DecisionPlayed, d2sfx.DecisionStolen, d2sfx.DecisionQueued, d2sfx.DecisionMerged:
	default:
		return nil
	}

	snd := &Sound{bank: s.bank, inst: inst, handle: r.Handle}
	s.sounds[snd] = struct{}{}

	return snd
}

// PlaySoundHandle plays a sound by sounds.txt handle
func (s *SoundEngine) PlaySoundHandle(handle string) *Sound {
	e, ok := s.asset.Records.Sound.Details[handle]
	if !ok {
		s.Warningf("unknown sound %q", handle)
		return nil
	}

	return s.PlaySoundID(e.Index)
}

// AutoSound resolves each comma-separated sound (a handle or a numeric index),
// plays it through the voice bank and logs the row, file, priority and the
// engine's decision. With OD2_AUTOTEST_MUTE set nothing is audible, so this
// verifies the rules without listening. Used by OD2_AUTOSOUND.
func (s *SoundEngine) AutoSound(spec string) {
	for _, name := range strings.Split(spec, ",") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		id, err := strconv.Atoi(name)
		if err != nil {
			e, ok := s.asset.Records.Sound.Details[name]
			if !ok {
				s.Infof("AUTOSOUND sound=%s found=false", name)
				continue
			}

			id = e.Index
		}

		inst := s.Bank().Play(d2sfx.Request{Index: id})
		r := inst.Report()
		row := s.Bank().Table().Get(id)
		group := 0

		if row != nil {
			group = row.GroupSize
		}

		s.Infof("AUTOSOUND sound=%s index=%d group=%d picked=%d handle=%s file=%q priority=%d decision=%s voice=%d victim=%q voices_in_use=%d",
			name, id, group, r.Picked, r.Handle, r.File, r.Priority, r.Decision, r.Voice, r.Victim, s.Bank().ActiveVoices())
	}
}

func (s *SoundEngine) commandPlaySoundID(args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid argument")
	}

	s.PlaySoundID(id)

	return nil
}

func (s *SoundEngine) commandPlaySound(args []string) error {
	s.PlaySoundHandle(args[0])

	return nil
}

func (s *SoundEngine) commandActiveSounds([]string) error {
	for sound := range s.sounds {
		s.Info(sound.String())
	}

	return nil
}
func (s *SoundEngine) commandKillSounds([]string) error {
	for sound := range s.sounds {
		sound.Stop()
	}

	return nil
}
