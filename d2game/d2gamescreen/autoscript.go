package d2gamescreen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2input/d2gamepad"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
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

// autoScriptGames counts the games of the process that ran initAutoScript (OD2_AUTOSCRIPT_<n>).
//
//nolint:gochecknoglobals // one counter per process
var autoScriptGames int

type autoScriptState struct {
	runner  *d2autoscript.Runner
	log     *logCapture
	elapsed float64
}

// initAutoScript parses OD2_AUTOSCRIPT and starts capturing the game log.
// A malformed script is reported and fails the run at once.
func (v *Game) initAutoScript() {
	spec := os.Getenv("OD2_AUTOSCRIPT")

	// OD2_AUTOSCRIPT_<n> is the script of the n-th game of the process (1 = first), for flows that play a hero,
	// save and exit, and play it again from the menus (the reload must not repeat the first game's script)
	autoScriptGames++
	if s, ok := os.LookupEnv("OD2_AUTOSCRIPT_" + strconv.Itoa(autoScriptGames)); ok {
		spec = s
	}
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

	// the game client logs what the other players do (PLAYER ADD / LEAVE / CAST, CHAT): the processes
	// of a network game wait for those lines with waitlog:
	if v.gameClient != nil {
		v.gameClient.SetLogTap(capture)
	}

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

	v.autoScriptHeal()
	s.runner.Advance(elapsed)
}

// autoScriptHeal is the "scripted strong hero" of the Hell scenarios: with OD2_AUTOSCRIPT_HEAL=1 the hero's life and
// mana are topped up on every update while the script runs (a test aid like the console command restorevitals; the hero
// still dies from a hit stronger than his whole life, and a hero that is already dead stays dead).
func (v *Game) autoScriptHeal() {
	p := v.localPlayer
	if os.Getenv("OD2_AUTOSCRIPT_HEAL") != "1" || p == nil || p.Stats == nil || p.Stats.Health <= 0 {
		return
	}

	p.Stats.Health, p.Stats.Mana = p.Stats.MaxHealth, p.Stats.MaxMana
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
	// a living entity wins over an object or item of the same name (Lut Gholein has "Jerhyn" objects as well
	// as the NPC, and the map's iteration order is random)
	var found d2interface.MapEntity

	for _, e := range h.v.gameClient.MapEngine.Entities() {
		if !strings.EqualFold(e.Label(), label) {
			continue
		}

		switch e.(type) {
		case *d2mapentity.Object, *d2mapentity.Item:
			if found == nil {
				found = e
			}
		default:
			found = e
		}
	}

	if found == nil {
		return fmt.Errorf("no NPC %q on this map", label)
	}

	nx, ny := found.GetPositionF()
	hx, hy := h.v.localPlayer.GetPositionF()
	h.v.Infof("AUTOSCRIPT move:npc %q is a %T at (%.1f,%.1f), hero at (%.1f,%.1f)", label, found, nx, ny, hx, hy)

	h.v.OnPlayerInteract(found)

	return nil
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

// HoldStart, HoldTick and HoldEnd implement d2autoscript.HoldHost.
func (h autoScriptHost) HoldStart(spec string) error { return h.v.gameControls.AutoHoldStart(spec) }

func (h autoScriptHost) HoldTick(elapsed float64) { h.v.gameControls.AutoHoldTick(elapsed) }

func (h autoScriptHost) HoldEnd() { h.v.gameControls.AutoHoldEnd() }

func (h autoScriptHost) Press(key string) error { return h.v.gameControls.AutoPress(key) }

// Pad implements d2autoscript.PadHost: the step drives the synthetic
// controller, which the input service merges with the real ones.
func (h autoScriptHost) Pad(op string) error {
	h.v.Infof("PAD step %s", op)

	if op == "state" {
		h.v.Infof("PAD state %s", h.v.gameControls.AutoPadState())

		return nil
	}

	return d2gamepad.DefaultSynthetic().Do(op)
}
