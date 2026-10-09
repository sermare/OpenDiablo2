package d2gamescreen

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2daynight"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const (
	ambientTestDelay        = 2.0  // seconds after the hero exists before OD2_AUTOAMBIENT starts
	ambientTestPhaseSeconds = 30.0 // seconds spent in each phase
)

// ambientTest is the state of the OD2_AUTOAMBIENT scenario.
type ambientTest struct {
	level    *d2records.LevelDetailRecord
	phases   []int
	phaseIdx int
	phaseFor float64
	elapsed  float64
	started  bool
	speed    float64
	secs     float64
}

// soundEnvForRegion maps a map region (the d2enum region ids match the
// LevelType column of Levels.txt) to a SoundEnviron row: the SoundEnv of the
// lowest-numbered level of that type. Regions without such a level keep the
// SoundEnv of the Levels.txt row with the same number, the previous behaviour.
func (v *Game) soundEnvForRegion(region d2enum.RegionIdType, fallback int) int {
	if v.regionEnvs == nil {
		v.regionEnvs = map[int]int{}

		levels := v.asset.Records.Level.Details
		ids := make([]int, 0, len(levels))

		for id := range levels {
			ids = append(ids, id)
		}

		sort.Ints(ids)

		for _, id := range ids {
			if _, seen := v.regionEnvs[levels[id].LevelType]; !seen {
				v.regionEnvs[levels[id].LevelType] = levels[id].SoundEnvironmentID
			}
		}
	}

	if env, ok := v.regionEnvs[int(region)]; ok {
		return env
	}

	return fallback
}

// ambientSpeed scales the ambient event timers (OD2_AUTOAMBIENT_SPEED).
func (v *Game) ambientSpeed() float64 {
	if v.ambientTest != nil && v.ambientTest.started {
		return v.ambientTest.speed
	}

	return 1
}

// resolveLevel finds a Levels.txt row by id or by name (Name or display name,
// ignoring case).
func (v *Game) resolveLevel(ref string) *d2records.LevelDetailRecord {
	levels := v.asset.Records.Level.Details

	if id, err := strconv.Atoi(ref); err == nil {
		return levels[id]
	}

	ids := make([]int, 0, len(levels))
	for id := range levels {
		ids = append(ids, id)
	}

	sort.Ints(ids)

	for _, id := range ids {
		l := levels[id]
		if strings.EqualFold(l.Name, ref) || strings.EqualFold(l.LevelDisplayName, ref) {
			return l
		}
	}

	return nil
}

// advanceAutoAmbient implements OD2_AUTOAMBIENT=<level id|name>: it puts the
// sound environment of that Levels.txt row in place (the hero stays where it
// is) and cycles through day phases, logging the selected SoundEnviron row,
// the day phase, the music track, the ambience loop and every scheduled and
// played ambient event (AUTOAMBIENT and AMBIENT lines; SOUNDAT lines show the
// volume and pan of each event). Settings:
//
//	OD2_AUTOAMBIENT_PHASES=2,4   day phases to visit (default 2,4: day then night)
//	OD2_AUTOAMBIENT_SECONDS=30   seconds per phase
//	OD2_AUTOAMBIENT_SPEED=1      multiplier on the event timers, to see more events
//
// OD2_AUTOTEST_MUTE keeps it silent and OD2_AUTOEXIT quits afterwards.
func (v *Game) advanceAutoAmbient(elapsed float64) {
	ref := os.Getenv("OD2_AUTOAMBIENT")
	if ref == "" || v.localPlayer == nil {
		return
	}

	t := v.ambientTest
	if t == nil {
		t = v.newAmbientTest(ref)
		v.ambientTest = t
	}

	if t.level == nil {
		return
	}

	t.elapsed += elapsed
	if t.elapsed < ambientTestDelay {
		return
	}

	if !t.started {
		t.started = true
		v.dayClock.frozen = true
		v.dayClock.env.SetPhaseAndTick(t.phases[0], 0)
		v.soundEnv.SetDayPhase(t.phases[0]) // before the environment starts, so its first line shows the phase
		v.soundEnv.SetEnv(t.level.SoundEnvironmentID)
		v.enterAmbientPhase(t)

		return
	}

	t.phaseFor += elapsed
	if t.phaseFor >= t.secs {
		t.phaseIdx++
		t.phaseFor = 0

		if t.phaseIdx >= len(t.phases) {
			v.Infof("AUTOAMBIENT summary level=%d(%s) env=%s phases=%v %s", t.level.ID, t.level.Name,
				v.soundEnv.Environment().Handle, t.phases, v.soundSummary())
			t.level = nil

			v.autoTestExit()

			return
		}

		v.enterAmbientPhase(t)
	}
}

func (v *Game) newAmbientTest(ref string) *ambientTest {
	t := &ambientTest{phases: []int{d2daynight.PhaseDay, d2daynight.PhaseNight4}, secs: ambientTestPhaseSeconds, speed: 1}

	// an unnamed level keeps the test inert, and the hero's own environment plays
	t.level = v.resolveLevel(ref)
	if t.level == nil {
		v.Errorf("AUTOAMBIENT: unknown level %q", ref)
		v.autoTestExit()

		return t
	}

	if list := os.Getenv("OD2_AUTOAMBIENT_PHASES"); list != "" {
		var phases []int

		for _, p := range strings.Split(list, ",") {
			if n, err := strconv.Atoi(strings.TrimSpace(p)); err == nil && n >= 0 && n < d2daynight.PhaseCount {
				phases = append(phases, n)
			}
		}

		if len(phases) > 0 {
			t.phases = phases
		}
	}

	if secs, err := strconv.ParseFloat(os.Getenv("OD2_AUTOAMBIENT_SECONDS"), 64); err == nil && secs > 0 {
		t.secs = secs
	}

	if sp, err := strconv.ParseFloat(os.Getenv("OD2_AUTOAMBIENT_SPEED"), 64); err == nil && sp > 0 {
		t.speed = sp
	}

	v.dayClock = newDayClock()

	v.Infof("AUTOAMBIENT start level=%d name=%q env_id=%d phases=%v seconds_per_phase=%.0f speed=%.1f",
		t.level.ID, t.level.Name, t.level.SoundEnvironmentID, t.phases, t.secs, t.speed)

	return t
}

func (v *Game) enterAmbientPhase(t *ambientTest) {
	phase := t.phases[t.phaseIdx]
	v.dayClock.env.SetPhaseAndTick(phase, 0)
	v.soundEnv.SetDayPhase(phase)

	sel := v.soundEnv.Selection()
	env := v.soundEnv.Environment()
	b := v.soundEngine.Bank().Table()

	name := func(i int) string {
		if row := b.Get(i); row != nil {
			return row.Handle
		}

		return "-"
	}

	v.Infof("AUTOAMBIENT phase=%d class=%d night=%v level=%d env=%s(%d) music=%s ambience=%s event=%s event_delay=%d indoors=%d "+
		"next_event_in=%.1fs", phase, d2daynight.Classify(phase), sel.Night, t.level.ID, env.Handle, env.Index,
		name(sel.Song), name(sel.Ambience), name(sel.Event), env.EventDelay, env.Indoors,
		v.soundEnv.NextEventIn()/d2audioTicksPerSecond)
}

// d2audioTicksPerSecond converts the environment's tick timers to seconds.
const d2audioTicksPerSecond = 25.0
