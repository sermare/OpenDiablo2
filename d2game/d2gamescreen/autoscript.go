package d2gamescreen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2autoscript"
)

const (
	autoScriptDelaySeconds = 4.0 // let the map and the hero load first
	autoScriptLogLimit     = 1 << 20
)

// logCapture keeps the tail of the game log so expect:log=... can search it.
type logCapture struct {
	mu    sync.Mutex
	buf   []byte
	total int // bytes ever written (the buffer keeps the tail)
}

func (c *logCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.buf = append(c.buf, p...)
	c.total += len(p)
	if len(c.buf) > autoScriptLogLimit {
		c.buf = c.buf[len(c.buf)-autoScriptLogLimit/2:]
	}

	return len(p), nil
}

// contains searches the captured log, ignoring the runner's own lines.
func (c *logCapture) contains(substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, line := range bytes.Split(c.buf, []byte("\n")) {
		if !bytes.Contains(line, []byte("AUTOSCRIPT")) && bytes.Contains(line, []byte(substr)) {
			return true
		}
	}

	return false
}

// mark returns a position in the log for containsSince.
func (c *logCapture) mark() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.total
}

// containsSince searches the lines written after the mark (ignoring the
// runner's own lines).
func (c *logCapture) containsSince(mark int, substr string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	start := mark - (c.total - len(c.buf)) // the mark's index in the buffer
	if start < 0 {
		start = 0
	}

	if start > len(c.buf) {
		return false
	}

	for _, line := range bytes.Split(c.buf[start:], []byte("\n")) {
		if !bytes.Contains(line, []byte("AUTOSCRIPT")) && bytes.Contains(line, []byte(substr)) {
			return true
		}
	}

	return false
}

type autoScriptState struct {
	runner  *d2autoscript.Runner
	log     *logCapture
	elapsed float64
}

// initAutoScript parses OD2_AUTOSCRIPT and starts capturing the game log.
// A malformed script is reported and fails the run at once.
func (v *Game) initAutoScript() {
	spec := os.Getenv("OD2_AUTOSCRIPT")
	if spec == "" && os.Getenv("OD2_AUTOSAVE") == "1" {
		// change the gold, then exit; the exit saves the hero back to a .d2s
		spec = "wait:1;say:setgold " + autosaveTestGold + ";wait:1;exit"
	}

	if spec == "" {
		return
	}

	steps, err := d2autoscript.Parse(spec)
	if err != nil {
		v.Errorf("AUTOSCRIPT invalid script: %v", err)
		v.Infof("AUTOSCRIPT RESULT FAIL")
		v.autoScriptExit(false)

		return
	}

	capture := &logCapture{}
	v.Logger.Writer = io.MultiWriter(v.Logger.Writer, capture)
	v.autoScript = &autoScriptState{log: capture}
	v.autoScript.runner = d2autoscript.NewRunner(steps, autoScriptHost{v})
}

func (v *Game) advanceAutoScript(elapsed float64) {
	s := v.autoScript
	if s == nil || s.runner.Done() || v.localPlayer == nil || v.gameControls == nil {
		return
	}

	s.elapsed += elapsed
	if s.elapsed < autoScriptDelaySeconds {
		return
	}

	s.runner.Advance(elapsed)
}

func (v *Game) autoScriptExit(pass bool) {
	if os.Getenv("OD2_AUTOEXIT") != "" {
		if p := v.localPlayer; p != nil {
			v.Infof("HERO state at exit: level=%d exp=%d skillpoints=%d statpoints=%d gold=%d life=%d/%d",
				p.Stats.Level, p.Stats.Experience, p.Stats.SkillPoints, p.Stats.StatsPoints, p.Gold,
				p.Stats.Health, p.Stats.MaxHealth)
		}

		v.saveBeforeExit()
		v.leaveNetworkGame()

		if pass {
			os.Exit(0)
		}

		os.Exit(1)
	}
}

// autoScriptHost adapts Game to d2autoscript.Host.
type autoScriptHost struct{ v *Game }

func (h autoScriptHost) Move(x, y float64) error {
	h.v.npcTarget = nil
	h.v.OnPlayerMove(x, y)

	return nil
}

func (h autoScriptHost) MoveToNPC(label string) error {
	for _, e := range h.v.gameClient.MapEngine.Entities() {
		if strings.EqualFold(e.Label(), label) {
			h.v.OnPlayerInteract(e)
			return nil
		}
	}

	return fmt.Errorf("no NPC %q on this map", label)
}

func (h autoScriptHost) Cast(skill string, x, y float64, hasXY bool) error {
	v := h.v

	var id = -1

	for _, rec := range v.asset.Records.Skill.Details {
		if strings.EqualFold(rec.Skill, skill) {
			id = rec.ID
			break
		}
	}

	if id < 0 {
		return fmt.Errorf("unknown skill %q", skill)
	}

	if !hasXY {
		pos := v.localPlayer.Position.World()
		x, y = pos.X(), pos.Y()
	}

	v.OnPlayerCast(id, x, y)

	return nil
}

func (h autoScriptHost) Panel(name string) error { return h.v.gameControls.AutoPanel(name) }

func (h autoScriptHost) Say(command string) error { return h.v.terminal.Execute(command) }

func (h autoScriptHost) LogContains(substr string) bool { return h.v.autoScript.log.contains(substr) }

func (h autoScriptHost) LogMark() int { return h.v.autoScript.log.mark() }

func (h autoScriptHost) LogContainsSince(mark int, substr string) bool {
	return h.v.autoScript.log.containsSince(mark, substr)
}

func (h autoScriptHost) Logf(format string, args ...interface{}) { h.v.Infof(format, args...) }

func (h autoScriptHost) Exit(pass bool) { h.v.autoScriptExit(pass) }

// Skill, Hotkey and Press implement d2autoscript.SkillHost.
func (h autoScriptHost) Skill(op, arg string) error { return h.v.gameControls.AutoSkill(op, arg) }

func (h autoScriptHost) Hotkey(key, skill string) error {
	return h.v.gameControls.AutoHotkey(key, skill)
}

func (h autoScriptHost) Click(spec string) error { return h.v.gameControls.AutoClick(spec) }

func (h autoScriptHost) Press(key string) error { return h.v.gameControls.AutoPress(key) }
