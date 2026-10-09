package d2gamescreen

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2cube"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2uber"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
)

// OD2_AUTOUBER=<rewards|travel|cube|uber>[,...] scripts the quest rewards, the
// act-travel hooks, the Horadric Cube recipes of the quests and the Pandemonium
// event on a running hero (the 3D levels of the uber areas are not needed: the
// area is announced like OD2_AUTOBOSS does). The hero cannot die.
//
// Log lines: "AUTOUBER start", "AUTOUBER step", "AUTOUBER expect <what> PASS|FAIL",
// "AUTOUBER RESULT PASS|FAIL"; the engine's own lines ("QUEST EFFECT reward",
// "REWARD", "CUBE", "UBER", "QUEST travel") carry the evidence.

const uberStepLimit = 60.0 // seconds a single step may wait

type uberAutoTest struct {
	steps    []uberStep
	cur      int
	started  bool
	stepAt   float64
	elapsed  float64
	failures []string
	spawned  int
}

type uberStep struct {
	name string
	fn   func(v *Game, t *uberAutoTest, since float64) bool
}

func (t *uberAutoTest) expect(v *Game, what string, ok bool) {
	if ok {
		v.Infof("AUTOUBER expect %s PASS", what)

		return
	}

	t.failures = append(t.failures, what)
	v.Infof("AUTOUBER expect %s FAIL", what)
}

func uberAutoNames() []string {
	spec := strings.TrimSpace(os.Getenv("OD2_AUTOUBER"))
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

// advanceUberTest runs the OD2_AUTOUBER scenario.
func (v *Game) advanceUberTest(elapsed float64) {
	names := uberAutoNames()
	if len(names) == 0 || v.localPlayer == nil || v.monsters == nil || v.gameControls == nil {
		return
	}

	t := v.uberTest
	if t == nil {
		if v.quests() == nil {
			return
		}

		t = &uberAutoTest{}

		for _, n := range names {
			switch n {
			case "rewards":
				t.steps = append(t.steps, v.rewardSteps()...)
			case "travel":
				t.steps = append(t.steps, v.travelSteps()...)
			case "cube":
				t.steps = append(t.steps, v.cubeSteps()...)
			case "uber":
				t.steps = append(t.steps, v.uberSteps()...)
			default:
				v.Errorf("AUTOUBER: unknown part %q", n)
			}
		}

		v.uberTest = t
		v.Infof("AUTOUBER start parts=%v steps=%d", names, len(t.steps))
	}

	if st := v.localPlayer.Stats; st != nil { // a spectator that cannot die
		st.Health = st.MaxHealth
	}

	t.elapsed += elapsed

	if t.cur >= len(t.steps) {
		return
	}

	if !t.started {
		t.started, t.stepAt = true, t.elapsed
		v.Infof("AUTOUBER step %d/%d: %s", t.cur+1, len(t.steps), t.steps[t.cur].name)
	}

	done := t.steps[t.cur].fn(v, t, t.elapsed-t.stepAt)

	if !done && t.elapsed-t.stepAt > uberStepLimit {
		t.expect(v, fmt.Sprintf("step %q finishes in time", t.steps[t.cur].name), false)

		done = true
	}

	if done {
		t.cur++
		t.started = false

		if t.cur >= len(t.steps) {
			v.finishUberTest(t)
		}
	}
}

func (v *Game) finishUberTest(t *uberAutoTest) {
	if len(t.failures) == 0 {
		v.Infof("AUTOUBER RESULT PASS")
	} else {
		v.Infof("AUTOUBER RESULT FAIL %d: %v", len(t.failures), t.failures)
	}

	v.autoTestExit()
}

// ---- rewards ----

func (v *Game) rewardSteps() []uberStep {
	fx := func(code string, value int) d2quest.Effect {
		return d2quest.Effect{Kind: d2quest.EffectReward, Code: code, Value: value, Note: "autotest"}
	}

	return []uberStep{
		{"rewards: stat points, Potion of Life, Malah's scroll", func(v *Game, t *uberAutoTest, _ float64) bool {
			st := v.localPlayer.Stats
			sp, life := st.StatsPoints, st.MaxHealth

			var fire int
			if st.Totals != nil {
				fire = st.Totals.ResistShown[0]
			}

			v.applyQuestEffects([]d2quest.Effect{fx("stat-points", 5), fx("life-boost", 20), fx("resist-bonus", 10)})

			t.expect(v, "stat points +5", st.StatsPoints == sp+5)
			t.expect(v, "Potion of Life raises max life", st.MaxHealth > life)
			t.expect(v, "Malah's scroll raises resistances", st.ResistBonus == 10 && (st.Totals == nil || st.Totals.ResistShown[0] > fire))

			return true
		}},
		{"rewards: Larzuk's sockets on a sword", func(v *Game, t *uberAutoTest, _ float64) bool {
			name, err := v.giveRewardTestItem("lsd", 2)
			v.Infof("AUTOUBER gave %q err=%v", name, err)
			t.expect(v, "a sword was given", err == nil)

			v.applyQuestEffects([]d2quest.Effect{fx("socket-quest", 1)})
			t.expect(v, "socket reward is pending", v.questRT.rewards.SocketPending == 1)

			msg, err := v.applyItemReward("socket")
			v.Infof("AUTOUBER socket result %q err=%v", msg, err)
			t.expect(v, "Larzuk sockets the sword", err == nil && v.questRT.rewards.SocketPending == 0)

			_, err = v.applyItemReward("socket")
			t.expect(v, "a spent reward cannot be used twice", err != nil)

			return true
		}},
		{"rewards: Anya personalises a magic helm", func(v *Game, t *uberAutoTest, _ float64) bool {
			_, err := v.giveRewardTestItem("cap", 4)
			t.expect(v, "a magic helm was given", err == nil)

			v.applyQuestEffects([]d2quest.Effect{fx("personalize", 1)})

			msg, err := v.applyItemReward("personalize")
			v.Infof("AUTOUBER personalize result %q err=%v", msg, err)
			t.expect(v, "Anya personalises the helm", err == nil && strings.Contains(msg, v.localPlayer.Name()))

			return true
		}},
		{"rewards: mercenaries, difficulty, end of game", func(v *Game, t *uberAutoTest, _ float64) bool {
			v.applyQuestEffects([]d2quest.Effect{fx("hire-ironwolves", 0), fx("hire-barbarians", 0), fx("unlock-difficulty", 0), fx("game-complete", 0)})

			rw := &v.questRT.rewards
			t.expect(v, "mercenary rewards recorded", rw.Hired["ironwolves"] && rw.Hired["barbarians"])
			t.expect(v, "difficulty unlock and game end recorded", rw.DifficultyUnlocked && rw.Complete)

			return true
		}},
		{"rewards: Hellforge drops (Hephasto, Mephisto) and the smashing", func(v *Game, t *uberAutoTest, _ float64) bool {
			v.questKillDrops("Hephasto the Armorer", 0)
			v.questKillDrops("Mephisto", 242)

			r := v.questRT
			r.g.Items["hfh"], r.g.Items["mss"] = 1, 1

			q := r.g.Quest(d2quest.QuestHellforge)
			q.Active, q.NotIntro = true, true
			r.g.Rec.SetSlot(q.Slot, 0)
			r.g.Rec.Set(q.Slot, d2quest.FlagStarted)

			v.questDispatch(d2quest.Event{Kind: d2quest.EvObjectOperated, Object: d2quest.ObjectHellforge, Level: 107})

			t.expect(v, "the smashed Hellforge completes its quest", r.g.Rec.Get(q.Slot, d2quest.FlagPrimaryGoal))

			return true
		}},
	}
}

// ---- travel ----

func (v *Game) travelSteps() []uberStep {
	return []uberStep{
		{"travel: the quest system follows act changes (Warriv, Meshif, portal, Tyrael)", func(v *Game, t *uberAutoTest, _ float64) bool {
			rec := v.questRT.g.Rec

			for _, c := range []struct {
				from, to int
				via      string
			}{{1, 2, "act:npc"}, {2, 3, "act:npc"}, {3, 4, "portal"}, {4, 5, "act:talk"}} {
				v.questActChange(c.from, c.to, c.via)
				t.expect(v, fmt.Sprintf("act %d is finished after the trip to act %d", c.from, c.to), rec.ActFinished(c.from))
			}

			// trips that are not forward act changes are ignored
			v.questActChange(5, 4, "act:npc")
			v.questActChange(1, 2, "load")

			return true
		}},
	}
}

// ---- cube ----

func (v *Game) cubeSteps() []uberStep {
	put := func(codes ...string) error {
		for _, c := range codes {
			if err := v.gameControls.GiveItemToCube(c); err != nil {
				return err
			}
		}

		return nil
	}

	return []uberStep{
		{"cube: Staff of Kings + Viper amulet -> Horadric Staff", func(v *Game, t *uberAutoTest, _ float64) bool {
			t.expect(v, "staff parts fit in the cube", put(d2cube.CodeStaffShaft, d2cube.CodeViper) == nil)

			code, err := v.transmuteCube()
			t.expect(v, "the staff is assembled", err == nil && code == d2cube.CodeStaff)
			t.expect(v, "the cube holds only the staff", strings.Join(v.gameControls.CubeCodes(), ",") == d2cube.CodeStaff)
			v.gameControls.CubeClear()

			return true
		}},
		{"cube: Khalim's Flail + Heart + Eye + Brain -> Khalim's Will", func(v *Game, t *uberAutoTest, _ float64) bool {
			t.expect(v, "Khalim's parts fit in the cube", put(d2cube.CodeFlail, d2cube.CodeHeart, d2cube.CodeEye, d2cube.CodeBrain) == nil)

			code, err := v.transmuteCube()
			t.expect(v, "the Will is assembled", err == nil && code == d2cube.CodeWill)
			v.gameControls.CubeClear()

			return true
		}},
		{"cube: a wrong mix is refused", func(v *Game, t *uberAutoTest, _ float64) bool {
			t.expect(v, "items fit", put(d2cube.CodeStaffShaft, d2cube.CodeFlail) == nil)

			_, err := v.transmuteCube()
			t.expect(v, "no recipe for the mix", err != nil)
			t.expect(v, "the refused items stay in the cube", len(v.gameControls.CubeCodes()) == 2)
			v.gameControls.CubeClear()

			return true
		}},
	}
}

// ---- Pandemonium ----

func (v *Game) uberSteps() []uberStep {
	put := func(codes ...string) error {
		for _, c := range codes {
			if err := v.gameControls.GiveItemToCube(c); err != nil {
				return err
			}
		}

		return nil
	}

	var areaBoss = map[int]string{d2uber.LevelMatronsDen: "uberandariel", d2uber.LevelForgottenSands: "uberduriel",
		d2uber.LevelFurnaceOfPain: "uberizual"}

	steps := []uberStep{}

	for _, lvl := range []int{d2uber.LevelMatronsDen, d2uber.LevelForgottenSands, d2uber.LevelFurnaceOfPain} {
		lvl := lvl
		area, _ := d2uber.AreaOf(lvl)

		steps = append(steps, uberStep{fmt.Sprintf("uber: three keys -> portal to %s, the boss, its organ", area.Name),
			func(v *Game, t *uberAutoTest, _ float64) bool {
				t.expect(v, "keys fit in the cube", put(d2cube.CodeKeyTerror, d2cube.CodeKeyHate, d2cube.CodeKeyDestr) == nil)

				_, err := v.transmuteCube()
				t.expect(v, "the keys open a portal", err == nil)
				t.expect(v, "the cube is empty after the transmutation", len(v.gameControls.CubeCodes()) == 0)

				v.questArea(lvl)
				v.uberEnter(lvl)

				t.expect(v, area.Boss.Name+" is alive", v.uberBossAlive(areaBoss[lvl]))

				n := v.uberKill(areaBoss[lvl])
				t.expect(v, area.Boss.Name+" dies", n == 1)

				return true
			}})
	}

	steps = append(steps, uberStep{"uber: three organs -> red portal to Tristram, the three bosses, the reward",
		func(v *Game, t *uberAutoTest, _ float64) bool {
			if t.spawned == 0 {
				t.expect(v, "organs fit in the cube", put(d2cube.CodeHorn, d2cube.CodeBaalEye, d2cube.CodeMephBrain) == nil)

				_, err := v.transmuteCube()
				t.expect(v, "the organs open the Tristram portal", err == nil)

				v.questArea(d2uber.LevelTristram)
				v.uberEnter(d2uber.LevelTristram)
				t.expect(v, "Uber Mephisto is alive", v.uberBossAlive("ubermephisto"))
				t.spawned = 1

				return false
			}

			// Diablo and Baal arrive on their timers; kill whoever is there
			for _, k := range []string{"ubermephisto", "uberdiablo", "uberbaal"} {
				v.uberKill(k)
			}

			if v.uber.ev.State() != "done" {
				return false
			}

			t.expect(v, "all three Tristram bosses were spawned", v.uberSpawnedKeys() >= 3)

			return true
		}})

	return steps
}

func (v *Game) uberMonsters(key string) []*d2mapentity.Monster {
	var out []*d2mapentity.Monster

	st := v.monsters.FindStat(key)
	if st == nil || v.uber == nil {
		return nil
	}

	for _, m := range v.uber.spawned {
		if m.Alive() && m.MonstatID() == st.ID {
			out = append(out, m)
		}
	}

	return out
}

func (v *Game) uberBossAlive(key string) bool { return len(v.uberMonsters(key)) > 0 }

func (v *Game) uberKill(key string) int {
	n := 0

	for _, m := range v.uberMonsters(key) {
		if v.monsters.KillMonster(m) {
			n++
		}
	}

	return n
}

// uberSpawnedKeys counts the distinct uber bosses spawned in Tristram.
func (v *Game) uberSpawnedKeys() int {
	seen := map[int]bool{}

	for _, m := range v.uber.spawned {
		for _, k := range []string{"ubermephisto", "uberdiablo", "uberbaal"} {
			if st := v.monsters.FindStat(k); st != nil && m.MonstatID() == st.ID {
				seen[st.ID] = true
			}
		}
	}

	return len(seen)
}
