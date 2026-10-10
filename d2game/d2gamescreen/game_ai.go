package d2gamescreen

import (
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

const (
	aiTestObserve  = 2.5 // seconds of the monster's own AI before the first state is injected
	aiTestFrames   = 100 // frames (4 s) a state lasts
	aiTestGap      = 3.0 // seconds after the state ends before the next one
	aiTestDefault  = 12.0
	aiTestBystands = 2 // hostile bystanders spawned for the monster-vs-monster states
)

// aiAutoTest is the state of the OD2_AUTOAI scenario.
type aiAutoTest struct {
	ref      string
	states   []d2monster.ForcedKind
	elapsed  float64
	spawned  bool
	subject  *d2mapentity.Monster
	nextIdx  int
	nextAt   float64
	endAt    float64
	counts   map[string]int
	lastKind string
	also     []*alsoSubject
}

// commandForceState is the debug console command
// "forcestate <monster id> <state> <frames>": it puts the forced AI state
// (fear, blind, taunt, confuse, attract, charm) on a monster for that many
// game frames (25 per second). The id is the number in the "MONSTER spawn"
// log lines (the AI's unit id), not the entity uuid.
func (v *Game) commandForceState(args []string) error {
	const usage = "usage: forcestate <monster id> <fear|blind|taunt|confuse|attract|charm> <frames>"

	if len(args) != 3 {
		return errors.New(usage)
	}

	d := v.monsterDirector()
	if d == nil {
		return errors.New("forcestate works in a game")
	}

	id, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		return errors.New(usage)
	}

	kind, ok := d2monster.ParseForced(args[1])
	if !ok {
		return errors.New(usage)
	}

	frames, err := strconv.Atoi(args[2])
	if err != nil || frames < 1 {
		return errors.New(usage)
	}

	label, err := d.ForceState(uint32(id), kind, frames, 0)
	if err != nil {
		return err
	}

	v.Infof("forcestate: monster %d is now %s for %d frames", id, label, frames)

	return nil
}

// parseAIAuto splits OD2_AUTOAI=<archetype|monster>[,state=fear|confuse|...]
// (several states may be joined with + or |).
func parseAIAuto(spec string) (ref string, states []d2monster.ForcedKind, err error) {
	parts := strings.Split(spec, ",")
	ref = strings.TrimSpace(parts[0])

	if ref == "" {
		return "", nil, errors.New("empty archetype or monster")
	}

	for _, p := range parts[1:] {
		p = strings.TrimSpace(p)

		if !strings.HasPrefix(p, "state=") {
			return "", nil, fmt.Errorf("unknown option %q", p)
		}

		val := strings.TrimPrefix(p, "state=")

		for _, w := range strings.FieldsFunc(val, func(r rune) bool { return r == '+' || r == '|' }) {
			k, ok := d2monster.ParseForced(w)
			if !ok {
				return "", nil, fmt.Errorf("unknown state %q", w)
			}

			states = append(states, k)
		}
	}

	return ref, states, nil
}

// advanceAITest implements OD2_AUTOAI=<archetype|monster>[,state=<fear|confuse|
// attract|charm|blind|taunt>[+...]]: it spawns one monster of the archetype (an
// AI name such as Vulture or Summoner) or of the monster class (a monstats id
// such as skeleton1) next to the hero, lets its own AI run, then injects each
// state with the "forcestate" console command (the same path a debug user
// takes) and logs the AI state transitions ("MONSTER aistate ... from= to=")
// and the actions. For the states that involve other monsters (confuse,
// attract, charm) hostile bystanders are spawned next to the subject. The hero
// stays put and cannot die. OD2_AUTOAI_SECONDS bounds the run; OD2_AUTOEXIT
// quits afterwards.
func (v *Game) advanceAITest(elapsed float64) {
	spec := os.Getenv("OD2_AUTOAI")
	if spec == "" || v.localPlayer == nil || v.monsters == nil {
		return
	}

	t := v.aiTest
	if t == nil {
		ref, states, err := parseAIAuto(spec)
		if err != nil {
			v.Errorf("AUTOAI: %v", err)
			v.autoTestExit()

			return
		}

		t = &aiAutoTest{ref: ref, states: states, counts: map[string]int{}}
		v.aiTest = t

		v.monsters.OnEvent = func(kind, _ string) { t.counts[kind]++ }
	}

	// the hero is only a spectator and a target
	if v.localPlayer.Stats != nil {
		v.localPlayer.Stats.Health = v.localPlayer.Stats.MaxHealth
	}

	t.elapsed += elapsed
	if t.elapsed < monsterTestDelay {
		return
	}

	if !t.spawned {
		t.spawned = true
		v.spawnAITest(t)

		return
	}

	if t.subject == nil {
		return
	}

	if t.nextIdx < len(t.states) && t.elapsed >= t.nextAt {
		kind := t.states[t.nextIdx]
		t.nextIdx++
		t.nextAt = t.elapsed + float64(aiTestFrames)/25 + aiTestGap

		cmd := fmt.Sprintf("forcestate %d %s %d", t.subject.Brain.ID, kind, aiTestFrames)
		v.Infof("AUTOAI inject: %s", cmd)

		if err := v.terminal.Execute(cmd); err != nil {
			v.Errorf("AUTOAI: %v", err)
		}

		t.lastKind = kind.String()
	}

	if t.elapsed < t.endAt {
		return
	}

	b := t.subject.Brain
	v.Infof("AUTOAI summary ref=%s monster=%s id=%d ai=%s alive=%v hp=%d/%d states=%d aistate_changes=%d "+
		"attacks=%d unit_fights=%d skills=%d aggro=%d deaths=%d", t.ref, t.subject.Label(), b.ID, b.Profile.AI,
		t.subject.Alive(), t.subject.Vitals.HP, t.subject.Vitals.MaxHP, t.counts["state"], t.counts["aistate"],
		v.monsters.Counters.Attacks, v.monsters.Counters.UnitFights, t.counts["skill"], v.monsters.Counters.Aggro,
		v.monsters.Counters.Deaths)

	v.summarizeAlso(t)

	t.endAt = math.MaxFloat64

	v.autoTestExit()
}

func (v *Game) spawnAITest(t *aiAutoTest) {
	stat := v.monsters.FindStat(t.ref)
	if stat == nil {
		stat = v.monsters.FindArchetype(t.ref)
	}

	if stat == nil {
		v.Errorf("AUTOAI: no monster or AI archetype %q", t.ref)
		v.autoTestExit()

		return
	}

	hx, hy := int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())

	m, err := v.monsters.SpawnNear(stat, hx+monsterTestRing, hy, 2)
	if err != nil {
		v.Errorf("AUTOAI: spawn failed: %v", err)
		v.autoTestExit()

		return
	}

	t.subject = m
	b := m.Brain

	seconds := aiTestDefault
	if n := len(t.states); n > 0 {
		seconds = aiTestObserve + float64(n)*(float64(aiTestFrames)/25+aiTestGap)
	}

	if s, err := strconv.ParseFloat(os.Getenv("OD2_AUTOAI_SECONDS"), 64); err == nil && s > 0 {
		seconds = s
	}

	t.nextAt = t.elapsed + aiTestObserve
	t.endAt = t.elapsed + seconds

	v.spawnAlso(t)

	v.Infof("AUTOAI start ref=%s monster=%s key=%s id=%d class=%d ai=%s implemented=%v states=%v hero=(%d,%d)",
		t.ref, m.Label(), stat.Key, b.ID, stat.ID, b.Profile.AI, b.Def != nil && b.Def.Implemented, t.states, hx, hy)

	// bystanders for the states in which monsters meet monsters
	for _, k := range t.states {
		if k != d2monster.ForcedConfuse && k != d2monster.ForcedAttract && k != d2monster.ForcedCharm {
			continue
		}

		bs := v.monsters.FindStat("skeleton1")
		if bs == nil {
			break
		}

		sx, sy := m.SubtilePos()

		for i := 0; i < aiTestBystands; i++ {
			if _, err := v.monsters.SpawnNear(bs, sx+4+3*i, sy+2, 2); err != nil {
				v.Infof("AUTOAI: bystander not spawned: %v", err)
			}
		}

		break
	}
}
