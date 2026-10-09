package d2gamescreen

import (
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

func akaraSounds() d2records.SoundDetails {
	s := d2records.SoundDetails{}
	for _, h := range []string{
		"akara_greeting_1", "akara_greeting_2", "akara_greeting_inactive_1", "akara_greeting_inactive_2",
		"akara_greeting_return", "akara_greeting_time_1", "akara_greeting_time_2", "akara_greeting_time_3",
	} {
		s[h] = &d2records.SoundDetailRecord{Handle: h}
	}

	s["akara_greeting_1"].GroupSize = 2
	s["akara_greeting_inactive_1"].GroupSize = 2

	return s
}

func TestLoadGreetingSet(t *testing.T) {
	set := loadGreetingSet(akaraSounds(), "akara")
	if len(set.greeting) != 2 || len(set.inactive) != 2 || len(set.ret) != 1 {
		t.Fatalf("unexpected set %+v", set)
	}
}

func TestPickGreetingReturnAndTime(t *testing.T) {
	set := loadGreetingSet(akaraSounds(), "akara")
	recent := map[string]string{}

	if got := pickGreeting(set, greetingReturn, 2, "", recent, func(int) int { return 0 }); got != "akara_greeting_return" {
		t.Fatalf("return: %s", got)
	}

	// rnd always 0: inactive chosen first (0 == 50% branch), then time (0 == 1/3 branch).
	if got := pickGreeting(set, greetingNormal, 1, "", recent, func(int) int { return 0 }); got != "akara_greeting_time_1" {
		t.Fatalf("morning: %s", got)
	}

	if got := pickGreeting(set, greetingNormal, 4, "", recent, func(int) int { return 0 }); got != "akara_greeting_time_3" {
		t.Fatalf("evening: %s", got)
	}
}

func TestPickGreetingAvoidsRepeat(t *testing.T) {
	set := loadGreetingSet(akaraSounds(), "akara")
	recent := map[string]string{}
	last := ""

	// rnd returns 1 so the 50% inactive branch and 1/3 time branch are skipped.
	for i := 0; i < 10; i++ {
		got := pickGreeting(set, greetingNormal, 2, last, recent, func(int) int { return 1 })
		if got == last {
			t.Fatalf("repeated %s", got)
		}

		last = got
	}
}

type fakePhase int

func (f fakePhase) Phase() int { return int(f) }

func TestGreetingUsesPhaseSource(t *testing.T) {
	set := loadGreetingSet(akaraSounds(), "akara")
	zero := func(int) int { return 0 }

	for phase, want := range map[int]string{
		0: "akara_greeting_time_3", 1: "akara_greeting_time_1", 2: "akara_greeting_time_2",
		3: "akara_greeting_time_2", 4: "akara_greeting_time_3", 5: "akara_greeting_time_3",
	} {
		var src dayPhaseSource = fakePhase(phase)
		if got := pickGreeting(set, greetingNormal, src.Phase(), "", map[string]string{}, zero); got != want {
			t.Errorf("phase %d: %s want %s", phase, got, want)
		}
	}
}

func TestDayClockAdvances(t *testing.T) {
	c := newDayClock()
	if c.Phase() != 1 {
		t.Fatalf("start phase %d", c.Phase())
	}

	// 241 ticks at 25/s is about 9.64 s: dawn becomes day.
	c.Advance(9.7)

	if c.Phase() != 2 {
		t.Errorf("phase after 9.7s = %d", c.Phase())
	}
}
