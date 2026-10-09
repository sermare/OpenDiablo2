package d2audio

import (
	"fmt"
	"math"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

type fakeSink struct {
	loops   []int
	stopped []int
	events  []int
}

func (f *fakeSink) loop(index int) func() {
	f.loops = append(f.loops, index)
	return func() { f.stopped = append(f.stopped, index) }
}

func (f *fakeSink) event(index int, _ func(int) int) string {
	f.events = append(f.events, index)
	return fmt.Sprint(index)
}

func (f *fakeSink) handle(index int) string { return fmt.Sprint("s", index) }

func testEnvs() map[int]*d2records.SoundEnvironRecord {
	return map[int]*d2records.SoundEnvironRecord{
		0: {Handle: "NONE", Index: 0, EventDelay: 250},
		2: {Handle: "WILD", Index: 2, Song: 4679, DayAmbience: 70, NightAmbience: 71, DayEvent: 192, NightEvent: 197, EventDelay: 250},
		4: {Handle: "CAVE", Index: 4, Song: 4657, DayAmbience: 54, NightAmbience: 54, DayEvent: 87, NightEvent: 87, EventDelay: 200, Indoors: 1},
		5: {Handle: "TOWN", Index: 5, Song: 4673, DayAmbience: 70, NightAmbience: 71},
	}
}

func newTestEnv(sink *fakeSink) *SoundEnvironment {
	var logs int

	e := newSoundEnvironment(sink, testEnvs(), func(n int) int { return n / 2 },
		func(string, ...interface{}) { logs++ })

	return &e
}

func TestSelectDayNight(t *testing.T) {
	envs := testEnvs()

	tests := []struct {
		env   int
		night bool
		want  Selection
	}{
		{2, false, Selection{Song: 4679, Ambience: 70, Event: 192}},
		{2, true, Selection{Song: 4679, Ambience: 71, Event: 197, Night: true}},
		{4, true, Selection{Song: 4657, Ambience: 54, Event: 87, Night: true}},
		{0, false, Selection{}},
	}

	for _, tc := range tests {
		if got := Select(envs[tc.env], tc.night); got != tc.want {
			t.Errorf("Select(env %d, night=%v) = %+v, want %+v", tc.env, tc.night, got, tc.want)
		}
	}

	if got := Select(nil, true); got.Song != 0 || !got.Night {
		t.Errorf("Select(nil) = %+v", got)
	}
}

func TestIsNight(t *testing.T) {
	want := map[int]bool{0: true, 1: false, 2: false, 3: false, 4: true, 5: true}
	for phase, night := range want {
		if IsNight(phase) != night {
			t.Errorf("IsNight(%d) = %v, want %v", phase, !night, night)
		}
	}
}

func TestNextEventTicksRange(t *testing.T) {
	for _, delay := range []int{0, 1, 2, 250, 800} {
		for r := 0; r < delay || r == 0; r += 7 {
			got := NextEventTicks(delay, func(n int) int {
				if r >= n {
					return n - 1
				}

				return r
			})

			lo, hi := delay/2, delay/2+delay
			if delay <= 1 {
				lo, hi = 1, 1
			}

			if got < lo || got > hi {
				t.Errorf("NextEventTicks(%d) = %d outside [%d,%d]", delay, got, lo, hi)
			}

			if delay == 0 {
				break
			}
		}
	}
}

func TestEventPositionStaysInsideRadius(t *testing.T) {
	const radius = 1000.0

	for a := 0; a < 1000; a += 37 {
		for d := 0; d < 1000; d += 41 {
			calls := 0
			x, y := EventPosition(10, -20, radius, func(int) int {
				calls++
				if calls == 1 {
					return a
				}

				return d
			})

			// the bank doubles y when measuring distance
			dist := math.Hypot(x-10, 2*(y+20))
			if dist < 0.15*radius-1e-6 || dist > 0.9*radius+1e-6 {
				t.Fatalf("angle %d dist %d: distance %.1f outside 15%%..90%% of %v", a, d, dist, radius)
			}
		}
	}
}

func TestEnvironmentSwitchesMusicAndAmbience(t *testing.T) {
	sink := &fakeSink{}
	e := newTestEnv(sink)

	e.SetEnv(2) // wilderness, day
	if want := []int{4679, 70}; fmt.Sprint(sink.loops) != fmt.Sprint(want) {
		t.Fatalf("loops %v, want %v", sink.loops, want)
	}

	e.SetEnv(2) // same env: nothing restarts
	if len(sink.loops) != 2 {
		t.Fatalf("re-entering the same env restarted sounds: %v", sink.loops)
	}

	e.SetDayPhase(4) // night: ambience swaps, music stays
	if want := []int{4679, 70, 71}; fmt.Sprint(sink.loops) != fmt.Sprint(want) {
		t.Fatalf("loops after dusk %v, want %v", sink.loops, want)
	}

	if fmt.Sprint(sink.stopped) != "[70]" {
		t.Fatalf("stopped %v, want [70]", sink.stopped)
	}

	e.SetDayPhase(5) // still night: no change
	if len(sink.loops) != 3 {
		t.Fatalf("night to night restarted sounds: %v", sink.loops)
	}

	e.SetEnv(4) // cave: music and ambience change (ambience 54 same day/night)
	if fmt.Sprint(sink.loops) != "[4679 70 71 4657 54]" {
		t.Fatalf("loops after entering the cave %v", sink.loops)
	}

	if got := e.Selection(); got.Event != 87 || !got.Night {
		t.Fatalf("cave selection %+v", got)
	}

	e.SetDayPhase(2) // day inside the cave: nothing changes (both columns equal) but events keep
	if len(sink.loops) != 5 {
		t.Fatalf("cave day/night changed loops: %v", sink.loops)
	}
}

func TestEnvironmentKeepsSharedLoops(t *testing.T) {
	sink := &fakeSink{}
	e := newTestEnv(sink)
	e.SetEnv(2)
	e.SetEnv(5) // town 'TOWN' has another song
	e.SetEnv(2)

	if fmt.Sprint(sink.loops) != "[4679 70 4673 4679]" {
		t.Fatalf("loops %v", sink.loops)
	}
}

func TestEnvironmentEvents(t *testing.T) {
	sink := &fakeSink{}
	e := newTestEnv(sink) // rnd(n) = n/2, so delay 250 -> 125 + 125 = 250 ticks = 10 s

	e.SetEnv(2)

	if got := e.NextEventIn(); got != 250 {
		t.Fatalf("first event in %v ticks, want 250", got)
	}

	e.Advance(9.9)
	if len(sink.events) != 0 {
		t.Fatalf("event fired early: %v", sink.events)
	}

	e.Advance(0.2)
	if fmt.Sprint(sink.events) != "[192]" {
		t.Fatalf("events %v, want the day event [192]", sink.events)
	}

	e.SetDayPhase(4)
	e.Advance(10.5)

	if fmt.Sprint(sink.events) != "[192 197]" {
		t.Fatalf("events %v, want day then night event", sink.events)
	}
}

func TestEnvironmentWithoutEventsStaysQuiet(t *testing.T) {
	sink := &fakeSink{}
	e := newTestEnv(sink)

	e.SetEnv(5) // no events
	e.Advance(1000)

	if len(sink.events) != 0 {
		t.Fatalf("events %v from an environment without any", sink.events)
	}
}

func TestEnvironmentUnknownEnvIgnored(t *testing.T) {
	sink := &fakeSink{}
	e := newTestEnv(sink)
	e.SetEnv(99)

	if len(sink.loops) != 0 {
		t.Fatalf("unknown env started %v", sink.loops)
	}
}

func TestSubtileToSound(t *testing.T) {
	x, y := SubtileToSound(0, 0)
	if x != 0 || y != 0 {
		t.Fatalf("origin maps to %v,%v", x, y)
	}

	// equal subtile steps along either axis are equally far once y is doubled
	ax, ay := SubtileToSound(5, 0)
	bx, by := SubtileToSound(0, 5)

	da, db := math.Hypot(ax, 2*ay), math.Hypot(bx, 2*by)
	if math.Abs(da-db) > 1e-9 {
		t.Fatalf("anisotropic: %v vs %v", da, db)
	}

	// 12 subtiles (the AUTOMONSTER ring) must be audible for Falloff 0 (400)
	mx, my := SubtileToSound(12, 0)
	if d := math.Hypot(mx, 2*my); d >= 400 {
		t.Fatalf("12 subtiles = %.0f units, beyond the 400 radius", d)
	}
}
