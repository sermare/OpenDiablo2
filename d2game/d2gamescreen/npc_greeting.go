package d2gamescreen

import (
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2daynight"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const (
	greetingMaxTries    = 20
	greetingTimeOneIn   = 3 // GREETING_TIME_n is used 1/3 of the time
	greetingInactiveOne = 2 // inactive group is used 1/2 of the time
)

// greetingSet is the per-NPC greeting data the real game keeps in its
// 24 byte records (render-sound.md, SOUND_PickNpcGreeting 0x4dd4c0): each
// group is a list of Sounds.txt handles (group members, Group Size wide).
type greetingSet struct {
	greeting  []string
	inactive  []string
	timeOfDay [3][]string // TIME_1 (morning), TIME_2 (day), TIME_3 (evening)
	ret       []string    // GREETING_RETURN
}

// greetingMode mirrors the picker's mode parameter.
type greetingMode int

const (
	greetingNormal greetingMode = 0
	greetingReturn greetingMode = 2
)

// dayPhaseSource supplies the game's day/night phase (0..5) so the greeting
// does not depend on the wall clock and stays testable.
type dayPhaseSource interface {
	Phase() int
}

// dayClock drives a d2daynight environment from frame time.
type dayClock struct {
	env *d2daynight.Env
	acc float64
}

func newDayClock() *dayClock {
	return &dayClock{env: d2daynight.NewCycling()}
}

// Advance converts elapsed seconds into environment ticks.
func (c *dayClock) Advance(elapsed float64) {
	c.acc += elapsed * d2daynight.TicksPerSecond
	n := int(c.acc)
	c.acc -= float64(n)
	c.env.Advance(n)
}

// Phase implements dayPhaseSource.
func (c *dayClock) Phase() int { return c.env.Phase() }

// timeGroupForPhase returns the index of the GREETING_TIME_n group for the
// environment phase: phase 1 -> TIME_1, phases 2 and 3 -> TIME_2, phases 0,
// 4 and 5 -> TIME_3.
func timeGroupForPhase(phase int) int {
	return int(d2daynight.Classify(phase))
}

// loadGreetingSet gathers the greeting groups of an NPC from Sounds.txt
// using handles like akara_greeting_1, akara_greeting_inactive_1,
// akara_greeting_time_1 and akara_greeting_return.
func loadGreetingSet(sounds d2records.SoundDetails, npc string) greetingSet {
	base := npc + "_greeting_"

	var set greetingSet

	set.greeting = groupMembers(sounds, base)
	set.inactive = groupMembers(sounds, base+"inactive_")

	for i := range set.timeOfDay {
		if _, ok := sounds[fmt.Sprintf("%stime_%d", base, i+1)]; ok {
			set.timeOfDay[i] = []string{fmt.Sprintf("%stime_%d", base, i+1)}
		}
	}

	if _, ok := sounds[base+"return"]; ok {
		set.ret = []string{base + "return"}
	}

	return set
}

// groupMembers returns the handles of the group whose head is prefix+"1".
// A head with Group Size n covers n consecutive rows (prefix1..prefixn).
func groupMembers(sounds d2records.SoundDetails, prefix string) []string {
	head, ok := sounds[prefix+"1"]
	if !ok {
		return nil
	}

	size := head.GroupSize
	if size < 1 {
		size = 1
	}

	members := make([]string, 0, size)

	for i := 1; i <= size; i++ {
		handle := fmt.Sprintf("%s%d", prefix, i)
		if _, found := sounds[handle]; found {
			members = append(members, handle)
		}
	}

	return members
}

// pickGroupVariant picks a member of a group while avoiding the member
// played last from the same group (SOUND_PickGroupVariant keeps a history).
func pickGroupVariant(group []string, recent map[string]string, rnd func(int) int) string {
	if len(group) == 0 {
		return ""
	}

	key := group[0]
	choice := group[rnd(len(group))]

	if len(group) > 1 && choice == recent[key] {
		choice = group[(indexOf(group, choice)+1)%len(group)]
	}

	recent[key] = choice

	return choice
}

func indexOf(list []string, s string) int {
	for i := range list {
		if list[i] == s {
			return i
		}
	}

	return 0
}

// pickGreeting is the port of SOUND_PickNpcGreeting: it returns the handle
// of the line to play, never the same as lastPlayed when there is a choice.
func pickGreeting(set greetingSet, mode greetingMode, phase int, lastPlayed string,
	recent map[string]string, rnd func(int) int) string {
	if mode == greetingReturn {
		return pickGroupVariant(set.ret, recent, rnd)
	}

	var result string

	for try := 0; try < greetingMaxTries; try++ {
		group := set.greeting

		if len(set.inactive) > 0 && rnd(greetingInactiveOne) == 0 {
			group = set.inactive
		}

		timeGroup := set.timeOfDay[timeGroupForPhase(phase)]
		if len(timeGroup) > 0 && (len(group) == 0 || rnd(greetingTimeOneIn) == 0) {
			group = timeGroup
		}

		result = pickGroupVariant(group, recent, rnd)

		if result != lastPlayed {
			break
		}
	}

	return result
}
