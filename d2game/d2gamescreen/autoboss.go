package d2gamescreen

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2boss"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2monster"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
)

// OD2_AUTOBOSS=<duriel|mephisto|diablo|baal>[,...] drives the boss encounter
// logic (d2common/d2boss) directly: the hero stands still (and cannot die), the
// scenario plays the triggers the hero would (an object operated, a level
// entered, the hero walking up to a sleeping boss), the manager's actions are
// carried out with real monsters of the Director (their own AIs run, the Baal
// wave AIs call back into the manager), the scenario kills the monsters, and
// the quest bits that the kills set in the quest record are logged. The 3D
// levels are not needed: the area is announced to the quest system with the
// real level ids.
//
// Log lines: "AUTOBOSS start", "BOSS ..." (manager decisions), "AUTOBOSS spawn",
// "AUTOBOSS object", "AUTOBOSS kill", "AUTOBOSS QUEST", "AUTOBOSS RESULT".
//
// Boss monsters are resolved by monstats class (the id the quest system
// uses); the followers of a group are capped at bossGroupCap so that a slow
// machine is not flooded.

const (
	bossGroupCap  = 3
	bossStepLimit = 150.0 // seconds a single step may wait
	bossHeroOff   = 12
)

type bossAutoTest struct {
	names   []string
	mgr     *d2boss.Manager
	started bool
	steps   []bossStep
	cur     int
	stepAt  float64
	elapsed float64
	frames  float64

	super    map[*d2mapentity.Monster]int
	spawned  []*d2mapentity.Monster
	objects  map[int]d2monster.Point
	pass     bool
	failures []string

	baseline map[string]uint16
	results  map[string]uint16
	wave     []*d2mapentity.Monster
	waveAt   float64
	kills    int
}

// bossStep is one scripted step; it returns true when it is done.
type bossStep struct {
	name string
	fn   func(v *Game, t *bossAutoTest, since float64) bool
}

func bossAutoNames() []string {
	spec := strings.TrimSpace(os.Getenv("OD2_AUTOBOSS"))
	if spec == "" {
		return nil
	}

	var out []string

	for _, n := range strings.Split(spec, ",") {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			out = append(out, n)
		}
	}

	return out
}

func (t *bossAutoTest) fail(v *Game, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	t.failures = append(t.failures, msg)
	v.Infof("AUTOBOSS FAIL %s", msg)
}

// advanceBossTest runs the OD2_AUTOBOSS scenario.
func (v *Game) advanceBossTest(elapsed float64) {
	names := bossAutoNames()
	if len(names) == 0 || v.localPlayer == nil || v.monsters == nil {
		return
	}

	t := v.bossTest
	if t == nil {
		if v.quests() == nil {
			return
		}

		t = v.startBossTest(names)
		v.bossTest = t
	}

	if v.localPlayer.Stats != nil { // a spectator that cannot die
		v.localPlayer.Stats.Health = v.localPlayer.Stats.MaxHealth
	}

	t.elapsed += elapsed
	t.frames += elapsed * 25

	if n := int(t.frames); n > 0 {
		t.frames -= float64(n)
		v.bossApply(t, t.mgr.Tick(n))
	}

	if t.cur >= len(t.steps) {
		return
	}

	if !t.started {
		t.started, t.stepAt = true, t.elapsed
		v.Infof("AUTOBOSS step %d/%d: %s", t.cur+1, len(t.steps), t.steps[t.cur].name)
	}

	since := t.elapsed - t.stepAt

	if t.steps[t.cur].fn(v, t, since) {
		t.cur++
		t.started = false

		if t.cur >= len(t.steps) {
			v.finishBossTest(t)
		}

		return
	}

	if since > bossStepLimit {
		t.fail(v, "step %q did not finish in %.0f s", t.steps[t.cur].name, bossStepLimit)
		t.cur++
		t.started = false

		if t.cur >= len(t.steps) {
			v.finishBossTest(t)
		}
	}
}

func (v *Game) heroSub() (int, int) {
	return int(v.localPlayer.Position.X()), int(v.localPlayer.Position.Y())
}

func (v *Game) startBossTest(names []string) *bossAutoTest {
	t := &bossAutoTest{
		names: names, super: map[*d2mapentity.Monster]int{}, objects: map[int]d2monster.Point{},
		baseline: map[string]uint16{}, results: map[string]uint16{},
	}

	t.mgr = d2boss.New(func(s string) { v.Infof("%s", s) })

	// the save may have the boss quests done already: start them from scratch
	for id, slot := range map[int]int{d2quest.QuestSevenTombs: d2quest.SlotSevenTombs, d2quest.QuestGuardian: d2quest.SlotGuardian,
		d2quest.QuestTerrorsEnd: d2quest.SlotTerrorsEnd, d2quest.QuestEveOfDestruction: d2quest.SlotEveOfDestruction} {
		v.questRT.g.Rec.SetSlot(slot, 0)

		if q := v.questRT.g.Quest(id); q != nil {
			q.NotIntro, q.Active = true, true
		}
	}

	// the Baal AIs call back into the manager
	v.monsters.SetBossHooks(d2monsters.BossHooks{
		Throne: func(b *d2monster.Brain, step d2monster.ThroneStep, wave int) bool {
			as, ok := t.mgr.ThroneStep(int(step), wave)
			v.bossApply(t, as)

			return ok
		},
		WaveCleared: func() bool { return t.mgr.WaveCleared() },
		NearestObject: func(b *d2monster.Brain, class, radius int) (d2monster.Point, int, bool) {
			p, ok := t.objects[class]
			if !ok {
				return d2monster.Point{}, 0, false
			}

			d := d2monster.EdgeDistance(b.X-p.X, b.Y-p.Y, 1)

			return p, d, d <= radius
		},
		Left: func(b *d2monster.Brain) {
			v.Infof("AUTOBOSS baal left the level through the portal")
			v.bossApply(t, t.mgr.BaalLeft())
		},
	})

	// kills feed the encounter logic after the quest system saw them
	prev := v.monsters.OnKill
	if prev == nil { // the quest system installs its hook lazily
		prev = v.onMonsterKilled
	}

	v.monsters.OnKill = func(ev d2monsters.KillEvent) {
		if prev != nil {
			prev(ev)
		}

		super := -1
		if s, ok := t.super[ev.Monster]; ok {
			super = s
		}

		t.kills++
		v.Infof("AUTOBOSS kill name=%q class=%d super=%d", ev.Label, ev.Class, super)
		v.bossApply(t, t.mgr.Killed(d2boss.Kill{Class: ev.Class, Super: super, Name: ev.Label}))
	}

	v.Infof("AUTOBOSS start bosses=%v hero=%v", names, fmt.Sprint(v.localPlayer.Position))

	for _, n := range names {
		switch n {
		case "duriel":
			t.steps = append(t.steps, v.durielSteps()...)
		case "mephisto":
			t.steps = append(t.steps, v.mephistoSteps()...)
		case "diablo":
			t.steps = append(t.steps, v.diabloSteps()...)
		case "baal":
			t.steps = append(t.steps, v.baalSteps()...)
		default:
			v.Errorf("AUTOBOSS: unknown boss %q", n)
		}
	}

	return t
}

// questBits reads the quest slot of a boss quest.
func (v *Game) questBits(slot int) uint16 {
	return v.questRT.g.Rec.Slot(slot)
}

func (v *Game) bossEnter(t *bossAutoTest, level int) {
	v.questArea(level)
	v.bossApply(t, t.mgr.Enter(level))
}

// bossApply carries out the manager's actions.
func (v *Game) bossApply(t *bossAutoTest, actions []d2boss.Action) {
	for _, a := range actions {
		switch a.Kind {
		case d2boss.ActSpawnMonster:
			v.bossSpawn(t, a)
		case d2boss.ActSpawnObject, d2boss.ActPortal:
			hx, hy := v.heroSub()
			t.objects[a.Class] = d2monster.Point{X: hx + 6, Y: hy - 6}
			v.Infof("AUTOBOSS object %s id=%d level=%d at=(%d,%d)", a.Name, a.Class, a.Level, hx+6, hy-6)
		case d2boss.ActObjectMode:
			v.Infof("AUTOBOSS object-mode %s id=%d", a.Name, a.Class)
		case d2boss.ActMessage:
			v.Infof("AUTOBOSS message %q msg=%d", a.Name, a.Msg)
		case d2boss.ActSpawnNPC:
			v.Infof("AUTOBOSS npc %s class=%d (not spawned in the autotest)", a.Name, a.Class)
		case d2boss.ActWake:
			for _, m := range t.spawned {
				if m.Alive() && m.MonstatID() == a.Class {
					if b := v.monsters.BrainOf(m); b != nil {
						b.Wake = 0
					}
				}
			}

			v.Infof("AUTOBOSS wake %s", a.Name)
		}
	}
}

func (v *Game) bossStat(a d2boss.Action) (key string, ok bool) {
	d := v.monsters
	if a.Key != "" {
		if st := d.FindStat(a.Key); st != nil {
			return st.Key, true
		}
	}

	if st := d.FindStat(strconv.Itoa(a.Class)); st != nil {
		return st.Key, true
	}

	return "", false
}

func (v *Game) bossSpawn(t *bossAutoTest, a d2boss.Action) {
	d := v.monsters

	key, ok := v.bossStat(a)
	if !ok {
		t.fail(v, "no monster for %s class=%d key=%q", a.Name, a.Class, a.Key)

		return
	}

	st := d.FindStat(key)
	hx, hy := v.heroSub()

	if a.Key == "baalcrabtostairs" { // the throne turns into him: the statue goes away
		for _, m := range t.spawned {
			if b := d.BrainOf(m); b != nil && m.Alive() && strings.EqualFold(b.Profile.AI, "BaalThrone") {
				d.Dismiss(b)
			}
		}
	}

	n := 1 + a.Group

	if a.Group > bossGroupCap {
		n = 1 + bossGroupCap
	}

	for i := 0; i < n; i++ {
		x, y := hx+bossHeroOff+(i%3)*3, hy+(i/3)*3-3

		m, err := d.SpawnNear(st, x, y, 3)
		if err != nil {
			t.fail(v, "spawn %s: %v", a.Name, err)

			return
		}

		t.spawned = append(t.spawned, m)

		if a.Super >= 0 && i == 0 {
			t.super[m] = a.Super
		}

		b := d.BrainOf(m)
		if a.Asleep && i == 0 && b != nil {
			b.Wake = 1 << 30
		}

		if i == 0 {
			ai := ""
			if b != nil {
				ai = b.Profile.AI
				// the throne statue must not see the hero (it would shoot instead of sending waves)
				if strings.EqualFold(ai, "BaalThrone") {
					b.Profile.AIDist = 3
				}
			}

			if a.Super >= d2boss.SuperBaalWave && a.Super < d2boss.SuperBaalWave+5 {
				defer t.mgr.WaveSpawned(n)
			}

			v.Infof("AUTOBOSS spawn name=%q boss=%s class=%d key=%s ai=%s super=%d group=%d asleep=%v", m.Label(), a.Name,
				st.ID, key, ai, a.Super, n-1, a.Asleep)
		}
	}
}

// bossKillAll kills the live monsters spawned for a boss (matching class when class >= 0).
func (v *Game) bossKill(t *bossAutoTest, match func(m *d2mapentity.Monster) bool) int {
	n := 0

	for _, m := range t.spawned {
		if m.Alive() && match(m) && v.monsters.KillMonster(m) {
			n++
		}
	}

	return n
}

func anyMonster(*d2mapentity.Monster) bool { return true }

func classIs(c int) func(m *d2mapentity.Monster) bool {
	return func(m *d2mapentity.Monster) bool { return m.MonstatID() == c }
}

func (v *Game) bossQuestCheck(t *bossAutoTest, boss string, slot int, extraMask uint16) {
	bits := v.questBits(slot)
	t.results[boss] = bits
	v.Infof("AUTOBOSS QUEST boss=%s slot=%d bits=0x%04x before=0x%04x primary_goal=%v reward_pending=%v extra=%v", boss, slot,
		bits, t.baseline[boss], bits&(1<<d2quest.FlagPrimaryGoal) != 0, bits&(1<<d2quest.FlagRewardPending) != 0,
		extraMask == 0 || bits&extraMask == extraMask)

	if bits&(1<<d2quest.FlagPrimaryGoal) == 0 {
		t.fail(v, "%s: the quest bit was not set by the kill (slot %d = 0x%04x)", boss, slot, bits)
	}
}

func (v *Game) finishBossTest(t *bossAutoTest) {
	v.Infof("AUTOBOSS states: %s", t.mgr.States())

	if len(t.failures) == 0 {
		v.Infof("AUTOBOSS RESULT PASS kills=%d spawned=%d", t.kills, len(t.spawned))
	} else {
		v.Infof("AUTOBOSS RESULT FAIL %d: %v", len(t.failures), t.failures)
	}

	v.autoTestExit()
}

func (v *Game) heroNearBoss(t *bossAutoTest, x, y int) { t.mgr.HeroX, t.mgr.HeroY = x, y }

// ---- Duriel ----

func (v *Game) durielSteps() []bossStep {
	return []bossStep{
		{"duriel: orifice without the staff, then with it", func(v *Game, t *bossAutoTest, since float64) bool {
			t.baseline["duriel"] = v.questBits(d2quest.SlotSevenTombs)
			v.bossEnter(t, 67) // Tal Rasha's tomb

			v.bossApply(t, t.mgr.Operate(d2boss.Operate{Object: d2boss.ObjOrifice}))
			v.bossApply(t, t.mgr.Operate(d2boss.Operate{Object: d2boss.ObjOrifice, HasStaff: true}))

			return true
		}},
		{"duriel: the portal opens", func(v *Game, t *bossAutoTest, since float64) bool {
			return since > 2.5
		}},
		{"duriel: lair entered, Duriel fights, is killed", func(v *Game, t *bossAutoTest, since float64) bool {
			if since == 0 {
				v.bossApply(t, t.mgr.Operate(d2boss.Operate{Object: d2boss.ObjDurielPortal}))
				v.bossEnter(t, d2boss.LevelDurielLair)
			}

			if since < 4 {
				return false
			}

			v.bossKill(t, anyMonster)

			return true
		}},
		{"duriel: Tyrael and the quest bit", func(v *Game, t *bossAutoTest, since float64) bool {
			if since < 9 {
				return false
			}

			v.bossQuestCheck(t, "duriel", d2quest.SlotSevenTombs, 1<<d2quest.FlagCustom1)

			return true
		}},
	}
}

// ---- Mephisto ----

func (v *Game) mephistoSteps() []bossStep {
	return []bossStep{
		{"mephisto: Durance of Hate 3 entered, he sleeps", func(v *Game, t *bossAutoTest, since float64) bool {
			t.baseline["mephisto"] = v.questBits(d2quest.SlotGuardian)

			me := t.mgr.Encounter("mephisto").(*d2boss.Mephisto)
			me.X, me.Y = 1000, 1000
			v.heroNearBoss(t, 1300, 1000) // far from the lair
			v.bossEnter(t, d2boss.LevelDurance3)

			return true
		}},
		{"mephisto: still asleep while the hero is far", func(v *Game, t *bossAutoTest, since float64) bool {
			if since < 2 {
				return false
			}

			v.Infof("AUTOBOSS mephisto state=%s", t.mgr.Encounter("mephisto").State())

			if t.mgr.Encounter("mephisto").State() != "asleep" {
				t.fail(v, "mephisto woke although the hero was far")
			}

			return true
		}},
		{"mephisto: the hero walks up, he wakes and fights", func(v *Game, t *bossAutoTest, since float64) bool {
			if since == 0 {
				v.heroNearBoss(t, 1010, 1000)
			}

			if since < 5 {
				return false
			}

			v.bossKill(t, anyMonster)

			return true
		}},
		{"mephisto: the portal and the quest bit", func(v *Game, t *bossAutoTest, since float64) bool {
			if since < 13 {
				return false
			}

			v.bossQuestCheck(t, "mephisto", d2quest.SlotGuardian, 0)

			return true
		}},
	}
}

// ---- Diablo ----

func (v *Game) diabloSteps() []bossStep {
	return []bossStep{
		{"diablo: the seals", func(v *Game, t *bossAutoTest, since float64) bool {
			t.baseline["diablo"] = v.questBits(d2quest.SlotTerrorsEnd)
			v.bossEnter(t, d2boss.LevelChaos)

			for _, o := range []int{d2boss.ObjSealPlainA, d2boss.ObjSealVizier, d2boss.ObjSealDeSeis, d2boss.ObjSealInfector} {
				v.bossApply(t, t.mgr.Operate(d2boss.Operate{Object: o}))
			}

			return true
		}},
		{"diablo: the seal bosses fight and die", func(v *Game, t *bossAutoTest, since float64) bool {
			if since < 5 {
				return false
			}

			v.bossKill(t, anyMonster)

			return true
		}},
		{"diablo: the last seal summons him once all three bosses are dead", func(v *Game, t *bossAutoTest, since float64) bool {
			if since == 0 {
				before := len(t.spawned)
				v.bossApply(t, t.mgr.Operate(d2boss.Operate{Object: d2boss.ObjSealPlainB}))
				v.Infof("AUTOBOSS diablo state=%s new_monsters=%d", t.mgr.Encounter("diablo").State(), len(t.spawned)-before)
			}

			if since < 6 {
				return false
			}

			return true
		}},
		{"diablo: Diablo fights and dies, the quest bit", func(v *Game, t *bossAutoTest, since float64) bool {
			if since < 4 {
				return false
			}

			v.bossKill(t, anyMonster)
			v.bossQuestCheck(t, "diablo", d2quest.SlotTerrorsEnd, 0)

			return true
		}},
	}
}

// ---- Baal ----

func (v *Game) baalSteps() []bossStep {
	return []bossStep{
		{"baal: the throne room, five waves (each killed 2 s after it appears)", func(v *Game, t *bossAutoTest, since float64) bool {
			if since == 0 {
				t.baseline["baal"] = v.questBits(d2quest.SlotEveOfDestruction)
				v.bossEnter(t, d2boss.LevelThrone)
			}

			// kill every wave monster two seconds after the wave came
			if len(t.wave) != len(t.spawned) {
				t.waveAt = t.elapsed
				t.wave = append(t.wave[:0], t.spawned...)
			}

			if t.elapsed-t.waveAt > 2 {
				v.bossKill(t, func(m *d2mapentity.Monster) bool {
					b := v.monsters.BrainOf(m)

					return b != nil && !strings.EqualFold(b.Profile.AI, "BaalThrone") && !strings.EqualFold(b.Profile.AI, "BaalToStairs")
				})
			}

			th := t.mgr.Encounter("baal").(*d2boss.Throne)

			return th.Wave >= 5 && th.Alive == 0
		}},
		{"baal: the throne morphs and walks into the Worldstone portal", func(v *Game, t *bossAutoTest, since float64) bool {
			st := t.mgr.Encounter("baal").State()
			if strings.HasPrefix(st, "baal-in-worldstone") {
				return true
			}

			return false
		}},
		{"baal: the Worldstone Chamber, Baal fights and dies, the quest bit", func(v *Game, t *bossAutoTest, since float64) bool {
			if since == 0 {
				v.bossEnter(t, d2boss.LevelWorldstone)
			}

			if since < 5 {
				return false
			}

			v.bossKill(t, func(m *d2mapentity.Monster) bool {
				b := v.monsters.BrainOf(m)

				return b != nil && strings.HasPrefix(strings.ToLower(b.Profile.AI), "baalcrab")
			})
			v.bossQuestCheck(t, "baal", d2quest.SlotEveOfDestruction, 0)

			return true
		}},
	}
}
