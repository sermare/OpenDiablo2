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
