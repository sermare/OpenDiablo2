// Package d2autoscript implements the OD2_AUTOSCRIPT scenario runner: a small,
// display-free state machine that drives the hero through a list of steps
// (walk, cast, open panels, console commands, log expectations) so a change can
// be tested without a mouse. The game supplies a Host that performs the actions.
package d2autoscript

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Kind is the type of a script step.
type Kind string

// Step kinds.
const (
	KindWait   Kind = "wait"
	KindMove   Kind = "move"
	KindCast   Kind = "cast"
	KindPanel  Kind = "panel"
	KindSay    Kind = "say"
	KindExpect Kind = "expect"
	KindExit   Kind = "exit"
	// KindUse walks to an object (by name or objects.txt id) and uses it.
	KindUse Kind = "use"
	// KindWaypoint travels to a waypoint level through the open waypoint panel.
	KindWaypoint Kind = "waypoint"
)

// ExpectLevelTimeout is how long (game seconds) an expect:level step waits for
// a level change in progress (fade out, load, fade in) before it fails.
const ExpectLevelTimeout = 4.0

// Step is one parsed script step.
type Step struct {
	Kind     Kind
	Seconds  float64 // wait
	X, Y     float64 // move, cast target (tile units)
	HasXY    bool    // an explicit target was given
	Arg      string  // skill name, panel name, console command, expected substring
	Level    int     // waypoint: target level; expect:level=: expected level
	HasLevel bool    // an expect step checks the level
	Text     string  // the step as written, for logging
}

// Panels accepted by the panel step.
var Panels = []string{"inventory", "character", "skills", "quest", "close"}

// LevelHost is implemented by hosts that support the use, waypoint and
// expect:level steps (it is separate so other hosts need not change).
type LevelHost interface {
	// Use walks to the object named (or with the objects.txt id) and operates it.
	Use(target string) error
	// Waypoint chooses a level in the open waypoint panel.
	Waypoint(level int) error
	// Level reports the current level id and hero position in tiles.
	Level() (level int, x, y float64)
	// Busy reports that the hero is walking to an object or a level change is
	// running; the runner holds the next step until it is over.
	Busy() bool
}

// BusyTimeout is the longest the runner waits for a busy host (game seconds).
const BusyTimeout = 30.0

// Host performs the side effects of steps. Methods must not block.
type Host interface {
	Move(x, y float64) error
	// MoveToNPC walks to the named NPC and opens its menu (as clicking would).
	MoveToNPC(label string) error
	Cast(skill string, x, y float64, hasXY bool) error
	Panel(name string) error
	Say(command string) error
	// LogContains reports whether the game log seen so far contains substr.
	// It must ignore the runner's own "AUTOSCRIPT ..." lines, or every
	// expect step would match itself.
	LogContains(substr string) bool
	Logf(format string, args ...interface{})
	Exit(pass bool)
}

// Parse splits a semicolon-separated script into steps.
func Parse(spec string) ([]Step, error) {
	var steps []Step

	for _, raw := range strings.Split(spec, ";") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		s, err := parseStep(raw)
		if err != nil {
			return nil, fmt.Errorf("step %d %q: %w", len(steps)+1, raw, err)
		}

		steps = append(steps, s)
	}

	if len(steps) == 0 {
		return nil, errors.New("empty script")
	}

	return steps, nil
}

func parseStep(raw string) (Step, error) {
	name, arg := raw, ""
	if i := strings.Index(raw, ":"); i >= 0 {
		name, arg = raw[:i], strings.TrimSpace(raw[i+1:])
	}

	s := Step{Kind: Kind(strings.ToLower(strings.TrimSpace(name))), Arg: arg, Text: raw}

	var err error

	switch s.Kind {
	case KindWait:
		s.Seconds, err = strconv.ParseFloat(arg, 64)
		if err != nil || s.Seconds < 0 {
			return s, errors.New("wait needs non-negative seconds")
		}
	case KindMove:
		if strings.HasPrefix(arg, "npc=") {
			// move:npc=<label> walks up to that NPC like a click would
			s.Arg = strings.TrimPrefix(arg, "npc=")
			if s.Arg == "" {
				return s, errors.New("move:npc= needs a name")
			}

			break
		}

		s.X, s.Y, err = parseXY(arg)
		s.HasXY = true
	case KindCast:
		skill, target := arg, ""
		if i := strings.LastIndex(arg, "@"); i >= 0 {
			skill, target = strings.TrimSpace(arg[:i]), arg[i+1:]
		}

		if skill == "" {
			return s, errors.New("cast needs a skill name")
		}

		s.Arg = skill
		if target != "" {
			s.X, s.Y, err = parseXY(target)
			s.HasXY = true
		}
	case KindPanel:
		s.Arg = strings.ToLower(arg)
		if !contains(Panels, s.Arg) {
			return s, fmt.Errorf("unknown panel (want %s)", strings.Join(Panels, "|"))
		}
	case KindSay:
		if arg == "" {
			return s, errors.New("say needs a command")
		}
	case KindUse:
		if arg == "" {
			return s, errors.New("use needs an object name or id")
		}
	case KindWaypoint:
		s.Level, err = strconv.Atoi(arg)
		if err != nil || s.Level <= 0 {
			return s, errors.New("waypoint needs a level id")
		}
	case KindExpect:
		if strings.HasPrefix(arg, "level=") {
			s.Level, err = strconv.Atoi(strings.TrimPrefix(arg, "level="))
			s.HasLevel = true

			if err != nil || s.Level <= 0 {
				return s, errors.New("expect:level= needs a level id")
			}

			break
		}

		if !strings.HasPrefix(arg, "log=") || len(arg) == len("log=") {
			return s, errors.New("expect needs log=<substring> or level=<id>")
		}

		s.Arg = strings.TrimPrefix(arg, "log=")
	case KindExit:
	default:
		return s, fmt.Errorf("unknown step %q", name)
	}

	return s, err
}

func parseXY(s string) (x, y float64, err error) {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return 0, 0, errors.New("want x,y")
	}

	if x, err = strconv.ParseFloat(strings.TrimSpace(parts[0]), 64); err != nil {
		return 0, 0, err
	}

	if y, err = strconv.ParseFloat(strings.TrimSpace(parts[1]), 64); err != nil {
		return 0, 0, err
	}

	return x, y, nil
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}

	return false
}

// Runner executes steps one per Advance call (waits take game time).
type Runner struct {
	steps   []Step
	host    Host
	next    int
	waiting float64
	// levelWait is how long the current expect:level step has been waiting.
	levelWait float64
	busyWait  float64
	failed    bool
	done      bool
}

// NewRunner creates a runner for the steps.
func NewRunner(steps []Step, host Host) *Runner {
	return &Runner{steps: steps, host: host}
}

// Done reports whether the script has finished (including the final result line).
func (r *Runner) Done() bool { return r.done }

// Failed reports whether any step failed.
func (r *Runner) Failed() bool { return r.failed }

// Advance moves the script forward by elapsed seconds.
func (r *Runner) Advance(elapsed float64) {
	if r.done {
		return
	}

	if r.waiting > 0 {
		r.waiting -= elapsed
		if r.waiting > 0 {
			return
		}

		r.waiting = 0
	}

	if r.next >= len(r.steps) {
		r.finish()
		return
	}

	if lh, ok := r.host.(LevelHost); ok && lh.Busy() && r.busyWait < BusyTimeout {
		r.busyWait += elapsed
		return
	}

	r.busyWait = 0

	s := r.steps[r.next]

	if s.Kind == KindExpect && s.HasLevel && r.levelWait < ExpectLevelTimeout && !r.levelIs(s.Level) {
		r.levelWait += elapsed // a level change is still running
		return
	}

	r.levelWait = 0
	r.next++
	r.host.Logf("AUTOSCRIPT step %d: %s", r.next, s.Text)

	if err := r.run(s); err != nil {
		r.failed = true
		r.host.Logf("AUTOSCRIPT step %d FAIL: %v", r.next, err)
	}

	if s.Kind == KindExit {
		r.finish()
	}
}

func (r *Runner) run(s Step) error {
	switch s.Kind {
	case KindWait:
		r.waiting = s.Seconds
		return nil
	case KindMove:
		if !s.HasXY {
			return r.host.MoveToNPC(s.Arg)
		}

		return r.host.Move(s.X, s.Y)
	case KindCast:
		return r.host.Cast(s.Arg, s.X, s.Y, s.HasXY)
	case KindPanel:
		return r.host.Panel(s.Arg)
	case KindSay:
		return r.host.Say(s.Arg)
	case KindUse:
		lh, ok := r.host.(LevelHost)
		if !ok {
			return errors.New("host does not support use")
		}

		return lh.Use(s.Arg)
	case KindWaypoint:
		lh, ok := r.host.(LevelHost)
		if !ok {
			return errors.New("host does not support waypoint")
		}

		return lh.Waypoint(s.Level)
	case KindExpect:
		if s.HasLevel {
			return r.expectLevel(s.Level)
		}

		if !r.host.LogContains(s.Arg) {
			return fmt.Errorf("log does not contain %q", s.Arg)
		}
	}

	return nil
}

func (r *Runner) finish() {
	r.done = true

	if r.failed {
		r.host.Logf("AUTOSCRIPT RESULT FAIL")
	} else {
		r.host.Logf("AUTOSCRIPT RESULT PASS")
	}

	r.host.Exit(!r.failed)
}

func (r *Runner) levelIs(want int) bool {
	lh, ok := r.host.(LevelHost)
	if !ok {
		return false
	}

	got, _, _ := lh.Level()

	return got == want
}

// expectLevel logs the level and hero position and fails if the level differs.
func (r *Runner) expectLevel(want int) error {
	lh, ok := r.host.(LevelHost)
	if !ok {
		return errors.New("host does not support expect:level")
	}

	got, x, y := lh.Level()
	r.host.Logf("AUTOSCRIPT level=%d hero=(%.1f,%.1f) want=%d", got, x, y, want)

	if got != want {
		return fmt.Errorf("level is %d, want %d", got, want)
	}

	return nil
}
