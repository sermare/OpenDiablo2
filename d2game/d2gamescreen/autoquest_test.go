package d2gamescreen

import (
	"fmt"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
)

// fakeHost runs the OD2_AUTOQUEST scripts against a plain quest system.
type fakeHost struct {
	g        *d2quest.Game
	area     int
	skill    int
	level    int
	rogues   bool
	imbue    bool
	foes     int
	log      []string
	exited   bool
	pass     bool
	saved    bool
	spoken   []int
	effects  map[d2quest.EffectKind]int
	progress *d2hero.HeroProgress
}

func newFakeHost(a *autoQuest, level int) *fakeHost {
	p := &d2hero.HeroProgress{}
	diff := 2
	a.resetRecord(p, diff)

	h := &fakeHost{progress: p, level: level, area: d2quest.LevelRogueEncampment, effects: map[d2quest.EffectKind]int{}}
	h.g = d2quest.New(p.QuestRecord(diff), &p.NPC, diff)
	h.g.Hero = d2quest.Hero{Class: d2quest.ClassSorceress, Level: level}
	h.g.Start()

	return h
}

func (h *fakeHost) Infof(format string, args ...interface{}) {
	h.log = append(h.log, fmt.Sprintf(format, args...))
}
func (h *fakeHost) Errorf(format string, args ...interface{}) {
	h.log = append(h.log, "ERROR "+fmt.Sprintf(format, args...))
}
func (h *fakeHost) Q() *d2quest.Game { return h.g }
func (h *fakeHost) Area() int        { return h.area }
func (h *fakeHost) SkillPoints() int { return h.skill }
func (h *fakeHost) HeroLevel() int   { return h.level }
func (h *fakeHost) RogueHire() bool  { return h.rogues }
func (h *fakeHost) ImbuePending() bool {
	return h.imbue
}
func (h *fakeHost) LogText(act, index int) string { return "" }
func (h *fakeHost) FoesAlive() int                { return h.foes }
func (h *fakeHost) Fight()                        {}
func (h *fakeHost) SpawnFoes(n int)               { h.foes = n }
func (h *fakeHost) Save()                         { h.saved = true }
func (h *fakeHost) Exit(pass bool)                { h.exited, h.pass = true, pass }

func (h *fakeHost) Apply(effects []d2quest.Effect) {
	for _, e := range effects {
		h.effects[e.Kind]++

		switch e.Kind {
		case d2quest.EffectSkillPoint:
			h.skill += e.Value
		case d2quest.EffectHireRogues:
			h.rogues = true
		case d2quest.EffectImbue:
			h.imbue = true
		}
	}
}

func (h *fakeHost) Move(area int) {
	from := h.area
	h.area = area
	h.Apply(h.g.Dispatch(d2quest.Event{Kind: d2quest.EvAreaChanged, OldLevel: from, NewLevel: area}))
}

func (h *fakeHost) Dispatch(e d2quest.Event) {
	h.g.Hero.Level = h.level
	h.Apply(h.g.Dispatch(e))
}

func (h *fakeHost) Tick(frames int) { h.Apply(h.g.Tick(frames)) }

// Talk is the Talk row: one spoken line, or nothing.
func (h *fakeHost) Talk(class int) []int {
	d := h.g.Activate(class)

	s, ok := d.NextSpoken(h.g)
	if !ok {
		return nil
	}

	snd, has := d2quest.SoundForMessage(s.Msg)
	h.log = append(h.log, fmt.Sprintf("QUEST SPEECH class=%d msg=%d mode=%d sound=%d has=%v", class, s.Msg, s.Mode, snd.Index, has))
	h.Apply(h.g.Hear(class, s.Msg))

	if _, more := h.g.Activate(class).NextSpoken(h.g); !more {
		h.Apply(h.g.Close(class))
	}

	h.spoken = append(h.spoken, s.Msg)

	return []int{s.Msg}
}

// runScenario plays a script to its end and returns the log.
func runScenario(t *testing.T, spec string, level int) *fakeHost {
	t.Helper()

	a := newAutoQuest(spec)
	if len(a.stages) == 0 {
		t.Fatalf("unknown scenario %q", spec)
	}

	h := newFakeHost(a, level)

	for i := 0; i < 10000 && !a.done; i++ {
		a.advance(h, 0.1)
	}

	if !a.done {
		t.Fatalf("%s: scenario did not finish (step %d of %d)", spec, a.idx, len(a.steps))
	}

	if !h.exited || !h.saved {
		t.Fatalf("%s: scenario did not save and exit", spec)
	}

	for _, l := range h.log {
		if strings.Contains(l, ": FAIL") || strings.HasPrefix(l, "ERROR") {
			t.Errorf("%s: %s", spec, l)
		}
	}

	if !h.pass || a.checks == 0 {
		t.Errorf("%s: result pass=%v checks=%d failed=%d", spec, h.pass, a.checks, a.failed)
	}

	return h
}

func TestAutoQuestScenarios(t *testing.T) {
	for _, spec := range []string{"den", "burial", "tools", "cain", "tower", "andariel", "radament", "act1", "1", "Den of Evil", "sun", "staff", "arcane", "summoner", "tombs", "act2"} {
		spec := spec

		t.Run(spec, func(t *testing.T) {
			h := runScenario(t, spec, 94)
			t.Logf("%d log lines, %d spoken messages", len(h.log), len(h.spoken))
		})
	}
}

func TestAutoQuestAct1PersistsToProgress(t *testing.T) {
	h := runScenario(t, "act1", 30)

	rec := h.progress.QuestRecord(2)
	for id := 1; id <= 6; id++ {
		if !rec.Get(id, d2quest.FlagRewardGranted) {
			t.Errorf("quest %d is not granted in the saved record: %#x", id, rec.Slot(id))
		}
	}

	if !rec.ActFinished(1) || !rec.AkaraRespecAvailable() {
		t.Error("act 1 finished word / respec flag missing")
	}

	if h.skill != 1 {
		t.Errorf("act 1 grants exactly one skill point (Den of Evil), got %d", h.skill)
	}

	if !h.rogues {
		t.Error("Kashya's rogues not hirable")
	}
}

func TestAutoQuestFailsBelowLevel8(t *testing.T) {
	// the Malus chest needs clvl 8: the script must report the failure, not hide it
	a := newAutoQuest("tools")
	h := newFakeHost(a, 5)

	for i := 0; i < 10000 && !a.done; i++ {
		a.advance(h, 0.1)
	}

	if !a.done || h.pass {
		t.Fatalf("a level 5 hero must fail the tools scenario (done=%v pass=%v)", a.done, h.pass)
	}
}

func TestAutoQuestUnknownName(t *testing.T) {
	a := newAutoQuest("nonsense")
	h := &fakeHost{}

	a.advance(h, 1)

	if !a.done || !h.exited || h.pass {
		t.Fatal("an unknown quest must fail at once")
	}
}
