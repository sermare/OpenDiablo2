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
	KindShot   Kind = "shot"
	KindExit   Kind = "exit"
	// KindUse walks to an object (by name or objects.txt id) and uses it.
	KindUse Kind = "use"
	// KindAutomap sets the automap: on, off, full, mini or stats (it runs the
	// "automap" console command).
	KindAutomap Kind = "automap"
	// KindSkill selects and uses skills and drives the skill popup:
	// skill:left=<name>, skill:right=<name>, skill:popup=left|right|close,
	// skill:hover=<name>, skill:click=<name>, skill:use=left|right, skill:spend=<name> and
	// skill:nospend=<name> (passes when the point is refused).
	KindSkill Kind = "skill"
	// KindHotkey assigns a skill to a hotkey: hotkey:F1=<name>[@left].
	KindHotkey Kind = "hotkey"
	// KindPress presses a skill hotkey: press:F1.
	KindPress Kind = "press"
	// KindWaitLog waits (up to WaitLogTimeout game seconds) until the game log
	// contains the text, then goes on; a timeout fails the step. It lets two
	// processes of a network game run a scenario in step: waitlog:<substring>.
	KindWaitLog Kind = "waitlog"
	// KindWaypoint travels to a waypoint level through the open waypoint panel.
	KindWaypoint Kind = "waypoint"
	// KindTravel travels to the town of an act through the act travel rules;
	// KindRefuse expects the rules to refuse that trip.
	KindTravel Kind = "travel"
	KindRefuse Kind = "refuse"
	// KindWalkTo walks to an exit (walkto:exit=<level>: a border, stairs or
	// cave entrance leading to that level, the level change included) or to
	// an object (walkto:object=<name>) and waits until the hero is there.
	KindWalkTo Kind = "walkto"
	// KindKill fights: kill:all[,<seconds>] every monster of the level,
	// kill:near=<tiles>[,<seconds>] those within that distance.
	KindKill Kind = "kill"
	// KindUntil waits for a log line: until:<substring>,<timeout seconds>.
	KindUntil Kind = "until"
	// KindLoot picks up the items lying within a radius: loot:<tiles>[,<seconds>].
	KindLoot Kind = "loot"
	// KindMenu picks a row of the open NPC menu (menu:Talk, menu:Trade, a topic text).
	KindMenu Kind = "menu"
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
	Op       string  // skill step: the operation (left, right, popup, ...); hotkey step: the key
	Level    int     // waypoint: target level; expect:level=: expected level
	HasLevel bool    // an expect step checks the level
	Text     string  // the step as written, for logging
	// Target is the second half of walkto (exit or object name) and the
	// kill mode ("all" or "near"); Radius is kill:near's distance in tiles.
	Target string
	Radius float64
}

// Panels accepted by the panel step.
var Panels = []string{"inventory", "character", "skills", "quest", "party", "close"}

// SkillOps are accepted by the skill step.
var SkillOps = []string{"left", "right", "popup", "hover", "click", "use", "spend", "nospend"}

// AutomapModes are accepted by the automap step.
var AutomapModes = []string{"on", "off", "toggle", "full", "mini", "stats",
	"fade", "nofade", "names", "nonames", "party", "noparty", "center", "nocenter"}

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

// TravelHost is implemented by hosts that support the travel and refuse steps.
type TravelHost interface {
	// Travel starts the trip to the town of the act (1..5) as the travel NPC or
	// portal of the hero's act would; it returns the rule's refusal as an error.
	Travel(act int) error
}

// PlayHost is implemented by hosts that can play: walk to exits and objects,
// fight, and choose NPC menu rows. Like LevelHost it is optional.
type PlayHost interface {
	// WalkToExit starts the walk to the exit towards a level; Busy stays true
	// until the hero arrived (or gave up).
	WalkToExit(level int) error
	// WalkToObject walks to the named object (any kind) and stops next to it.
	WalkToObject(name string) error
	// Menu chooses a row of the open NPC menu by its label or action.
	Menu(row string) error
	// Kill makes the hero fight; radius <= 0 means every monster of the level.
	// Busy stays true until nothing is left to fight or seconds ran out.
	Kill(radius, seconds float64) error
}

// NamedKillHost is implemented by hosts that can fight the monsters of one name (kill:name=).
type NamedKillHost interface {
	// KillNamed makes the hero fight every living monster whose name contains the text (wherever it is in the level)
	// until none is left or the seconds ran out; Busy stays true meanwhile.
	KillNamed(name string, seconds float64) error
}

// SkillHost is implemented by hosts that support the skill, hotkey and press
// steps (separate so other hosts need not change).
type SkillHost interface {
	// Skill runs a skill step (see KindSkill) with its operation and argument.
	Skill(op, arg string) error
	// Hotkey puts a skill (name, optionally "@left") on the key ("F1").
	Hotkey(key, skill string) error
	// Press presses the hotkey ("F1").
	Press(key string) error
}

// LootHost is implemented by hosts that can pick up ground items.
type LootHost interface {
	// Loot walks to and picks up every item within radius tiles of the hero;
	// Busy stays true until none is left (or seconds ran out).
	Loot(radius, seconds float64) error
}

// LogMarkHost is implemented by hosts that can tell which log lines are new:
// until: then only accepts lines logged after the step started (otherwise an
// until:NPC menu opened would be satisfied by the menu of ten steps ago).
type LogMarkHost interface {
	// LogMark returns a position in the log.
	LogMark() int
	// LogContainsSince reports whether a line logged after the mark contains substr.
	LogContainsSince(mark int, substr string) bool
}

// WaitLogTimeout is the longest a waitlog step waits (game seconds).
const WaitLogTimeout = 90.0

// BusyTimeout is the longest the runner waits for a busy host (game seconds). It must exceed the longest play
// step a script asks for (kill:all,200 ...): at 120 s the runner went on while the hero was still fighting, so
// a check placed after the kill ran in the middle of it.
const BusyTimeout = 400.0

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
	case KindSkill:
		i := strings.Index(arg, "=")
		if i <= 0 || i == len(arg)-1 {
			return s, errors.New("skill needs <op>=<value>")
		}

		s.Op, s.Arg = strings.ToLower(strings.TrimSpace(arg[:i])), strings.TrimSpace(arg[i+1:])
		if !contains(SkillOps, s.Op) {
			return s, fmt.Errorf("unknown skill op (want %s)", strings.Join(SkillOps, "|"))
		}
	case KindHotkey:
		i := strings.Index(arg, "=")
		if i <= 0 || i == len(arg)-1 {
			return s, errors.New("hotkey needs <key>=<skill>")
		}

		s.Op, s.Arg = strings.TrimSpace(arg[:i]), strings.TrimSpace(arg[i+1:])
	case KindPress:
		if arg == "" {
			return s, errors.New("press needs a key")
		}

		s.Op = arg
	case KindAutomap:
		s.Arg = strings.ToLower(arg)
		if !contains(AutomapModes, s.Arg) {
			return s, fmt.Errorf("unknown automap mode (want %s)", strings.Join(AutomapModes, "|"))
		}
	case KindSay:
		if arg == "" {
			return s, errors.New("say needs a command")
		}
	case KindShot:
		// shot:<file>.png saves a screenshot of the window (the console's capframe)
		if arg == "" || strings.ContainsAny(arg, " \t") {
			return s, errors.New("shot needs a file name without spaces")
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
	case KindTravel, KindRefuse:
		s.Level, err = strconv.Atoi(arg)
		if err != nil || s.Level < 1 || s.Level > 5 {
			return s, errors.New(string(s.Kind) + " needs an act 1..5")
		}
	case KindWalkTo:
		switch {
		case strings.HasPrefix(arg, "exit="):
			s.Target = "exit"
			s.Level, err = strconv.Atoi(strings.TrimPrefix(arg, "exit="))

			if err != nil || s.Level <= 0 {
				return s, errors.New("walkto:exit= needs a level id")
			}
		case strings.HasPrefix(arg, "object=") && len(arg) > len("object="):
			s.Target, s.Arg = "object", strings.TrimPrefix(arg, "object=")
		default:
			return s, errors.New("walkto needs exit=<level> or object=<name>")
		}
	case KindKill:
		err = parseKill(&s, arg)
	case KindUntil:
		i := strings.LastIndex(arg, ",")
		if i <= 0 {
			return s, errors.New("until needs <log substring>,<timeout seconds>")
		}

		s.Arg = strings.TrimSpace(arg[:i])
		s.Seconds, err = strconv.ParseFloat(strings.TrimSpace(arg[i+1:]), 64)

		if err != nil || s.Seconds <= 0 || s.Arg == "" {
			return s, errors.New("until needs <log substring>,<positive timeout seconds>")
		}
	case KindMenu:
		if arg == "" {
			return s, errors.New("menu needs a row name")
		}
	case KindLoot:
		err = parseLoot(&s, arg)
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
	case KindWaitLog:
		if arg == "" {
			return s, errors.New("waitlog needs a substring")
		}
	case KindExit:
	default:
		return s, fmt.Errorf("unknown step %q", name)
	}

	return s, err
}

// parseLoot reads "<tiles>[,seconds]".
func parseLoot(s *Step, arg string) error {
	rad, secs := arg, ""
	if i := strings.Index(arg, ","); i >= 0 {
		rad, secs = strings.TrimSpace(arg[:i]), strings.TrimSpace(arg[i+1:])
	}

	r, err := strconv.ParseFloat(rad, 64)
	if err != nil || r <= 0 {
		return errors.New("loot needs a radius in tiles")
	}

	s.Radius, s.Seconds = r, DefaultLootSeconds

	if secs != "" {
		if s.Seconds, err = strconv.ParseFloat(secs, 64); err != nil || s.Seconds <= 0 {
			return errors.New("loot: bad timeout")
		}
	}

	return nil
}

// DefaultLootSeconds is how long a loot step runs unless it says otherwise.
const DefaultLootSeconds = 60.0

// parseKill reads "all[,seconds]", "near=<tiles>[,seconds]" or "name=<text>[,seconds]" (the monsters whose name
// contains the text, anywhere in the level: a quest boss such as "Shenk the Overseer").
func parseKill(s *Step, arg string) error {
	what, secs := arg, ""
	if i := strings.Index(arg, ","); i >= 0 {
		what, secs = strings.TrimSpace(arg[:i]), strings.TrimSpace(arg[i+1:])
	}

	s.Seconds = DefaultKillSeconds

	if secs != "" {
		v, err := strconv.ParseFloat(secs, 64)
		if err != nil || v <= 0 {
			return errors.New("kill: bad timeout")
		}

		s.Seconds = v
	}

	switch {
	case what == "all":
		s.Target = "all"
	case strings.HasPrefix(what, "near="):
		r, err := strconv.ParseFloat(strings.TrimPrefix(what, "near="), 64)
		if err != nil || r <= 0 {
			return errors.New("kill:near= needs a distance in tiles")
		}

		s.Target, s.Radius = "near", r
	case strings.HasPrefix(what, "name=") && len(what) > len("name="):
		s.Target, s.Arg = "name", strings.TrimPrefix(what, "name=")
	default:
		return errors.New("kill needs all, near=<tiles> or name=<text>")
	}

	return nil
}

// DefaultKillSeconds is how long a kill step fights unless it says otherwise.
const DefaultKillSeconds = 120.0

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
	logWait   float64
	failed    bool
	done      bool
	// until is the waiting until: step and how long it has waited
	until      *Step
	untilSince float64
	untilMark  int
	// actionMark is the log position at the start of the last step that did something
	actionMark int
	markInit   bool
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

	if lm, ok := r.host.(LogMarkHost); ok && !r.markInit {
		r.markInit, r.actionMark = true, lm.LogMark() // the log before the script started does not count
	}

	if r.until != nil {
		r.advanceUntil(elapsed)
		return
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

	if s.Kind == KindWaitLog && r.logWait < WaitLogTimeout && !r.host.LogContains(s.Arg) {
		r.logWait += elapsed
		return
	}

	r.levelWait = 0
	r.next++
	r.host.Logf("AUTOSCRIPT step %d: %s", r.next, s.Text)

	// an until: step looks for lines logged since the last step that did
	// something (not a wait or a check): the action's own lines come before
	// the until step starts
	if lm, ok := r.host.(LogMarkHost); ok && s.Kind != KindWait && s.Kind != KindExpect && s.Kind != KindUntil {
		r.actionMark = lm.LogMark()
	}

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
	case KindShot:
		return r.host.Say("capframe " + s.Arg)
	case KindAutomap:
		return r.host.Say("automap " + s.Arg)
	case KindUse:
		lh, ok := r.host.(LevelHost)
		if !ok {
			return errors.New("host does not support use")
		}

		return lh.Use(s.Arg)
	case KindSkill, KindHotkey, KindPress:
		sh, ok := r.host.(SkillHost)
		if !ok {
			return errors.New("host does not support skill steps")
		}

		switch s.Kind {
		case KindSkill:
			return sh.Skill(s.Op, s.Arg)
		case KindHotkey:
			return sh.Hotkey(s.Op, s.Arg)
		default:
			return sh.Press(s.Op)
		}
	case KindWaypoint:
		lh, ok := r.host.(LevelHost)
		if !ok {
			return errors.New("host does not support waypoint")
		}

		return lh.Waypoint(s.Level)
	case KindTravel, KindRefuse:
		th, ok := r.host.(TravelHost)
		if !ok {
			return errors.New("host does not support travel")
		}

		err := th.Travel(s.Level)
		if s.Kind == KindRefuse {
			if err == nil {
				return fmt.Errorf("travel to act %d was allowed, expected a refusal", s.Level)
			}

			return nil
		}

		return err
	case KindWalkTo, KindKill, KindMenu:
		return r.play(s)
	case KindLoot:
		lh, ok := r.host.(LootHost)
		if !ok {
			return errors.New("host does not support loot")
		}

		return lh.Loot(s.Radius, s.Seconds)
	case KindUntil:
		st := s
		r.until, r.untilSince = &st, 0

		if _, ok := r.host.(LogMarkHost); ok {
			r.untilMark = r.actionMark
		} else if r.host.LogContains(s.Arg) {
			r.until = nil
		}

		return nil
	case KindWaitLog:
		waited := r.logWait
		r.logWait = 0

		if !r.host.LogContains(s.Arg) {
			return fmt.Errorf("log did not contain %q within %.0fs", s.Arg, waited)
		}

		return nil
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

func (r *Runner) play(s Step) error {
	ph, ok := r.host.(PlayHost)
	if !ok {
		return fmt.Errorf("host does not support %s", s.Kind)
	}

	switch s.Kind {
	case KindWalkTo:
		if s.Target == "exit" {
			return ph.WalkToExit(s.Level)
		}

		return ph.WalkToObject(s.Arg)
	case KindKill:
		if s.Target == "name" {
			nh, ok := r.host.(NamedKillHost)
			if !ok {
				return fmt.Errorf("host does not support kill:name=")
			}

			return nh.KillNamed(s.Arg, s.Seconds)
		}

		return ph.Kill(s.Radius, s.Seconds)
	default:
		return ph.Menu(s.Arg)
	}
}

// advanceUntil waits for the log line of an until: step.
func (r *Runner) advanceUntil(elapsed float64) {
	s := r.until

	seen := false

	if lm, ok := r.host.(LogMarkHost); ok {
		seen = lm.LogContainsSince(r.untilMark, s.Arg)
	} else {
		seen = r.host.LogContains(s.Arg)
	}

	if seen {
		r.until = nil
		return
	}

	if r.untilSince += elapsed; r.untilSince >= s.Seconds {
		r.until = nil
		r.failed = true
		r.host.Logf("AUTOSCRIPT step %d FAIL: no log line containing %q within %.0f s", r.next, s.Arg, s.Seconds)
	}
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
