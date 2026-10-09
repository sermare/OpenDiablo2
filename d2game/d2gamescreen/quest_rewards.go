package d2gamescreen

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2cube"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2reward"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
)

// Quest rewards, quest drops, the Horadric Cube recipes of the quests and the
// act-travel hooks. The rules are pure packages (d2common/d2reward, d2cube,
// d2level/acttravel.go); this file applies them to the running hero.

// applyQuestReward does the rewards of the later acts. Stat points, the Potion
// of Life and Malah's scroll change the hero at once; Larzuk's sockets and
// Anya's personalisation wait for an item (rewarditem command or the autotest);
// the mercenary rewards are recorded (the hire UI reads them);
// the difficulty unlock and the end of the game are logged (the unlock itself
// comes from the quest bits, see d2difficulty).
func (v *Game) applyQuestReward(e d2quest.Effect) {
	r := v.questRT
	st := v.localPlayer.Stats
	o := r.rewards.Apply(e)

	note := "applied"

	switch {
	case o.Log != "" && strings.Contains(o.Log, "not known"):
		note = "unknown reward, ignored"
	case o.StatPoints != 0:
		st.StatsPoints += o.StatPoints
		note = fmt.Sprintf("stat points total=%d", st.StatsPoints)
	case o.LifeBonus != 0:
		st.LifeBonus += o.LifeBonus
		before := st.MaxHealth

		v.recalcHero()
		st.Health += st.MaxHealth - before

		if st.Health > st.MaxHealth {
			st.Health = st.MaxHealth
		}

		note = fmt.Sprintf("max life %d -> %d", before, st.MaxHealth)
	case o.ResistBonus != 0:
		st.ResistBonus += o.ResistBonus
		v.recalcHero()

		note = "resistances +" + fmt.Sprint(o.ResistBonus)
		if st.Totals != nil {
			note = fmt.Sprintf("resistances +%d now fire=%d cold=%d light=%d poison=%d", o.ResistBonus,
				st.Totals.ResistShown[0], st.Totals.ResistShown[1], st.Totals.ResistShown[2], st.Totals.ResistShown[3])
		}
	case o.PendingSocket:
		note = "Larzuk waits for an item (rewarditem socket)"

		v.gameControls.Speech.Notice("Larzuk can add sockets: use the console command rewarditem socket", questNoticeSeconds)
	case o.PendingPersonalize:
		note = "Anya waits for an item (rewarditem personalize)"

		v.gameControls.Speech.Notice("Anya can personalize an item: use the console command rewarditem personalize", questNoticeSeconds)
	case o.Hire != "":
		note = "mercenaries of " + o.Hire + " are hirable"
	case o.UnlockDifficulty:
		note = "next difficulty unlocked by the quest bits"
	case o.GameComplete:
		note = "game complete"
	}

	v.Infof("QUEST EFFECT reward %s value=%d (%s) %s", e.Code, e.Value, e.Note, note)
}

// recalcHero recomputes the hero's derived stats after a permanent bonus.
func (v *Game) recalcHero() {
	if st := v.localPlayer.Stats; st != nil && st.Recalc != nil {
		st.Recalc()
	}
}

// questDifficulty is the difficulty the hero plays in (0..2).
func (v *Game) questDifficulty() int {
	if v.questRT != nil && v.questRT.g != nil && v.questRT.g.Difficulty >= 0 {
		return v.questRT.g.Difficulty
	}

	return int(v.gameClient.Difficulty)
}

// applyItemReward spends a pending item reward on an item of the hero (the
// cursor first, then the inventory) and returns what happened. kind is
// "socket" (Larzuk) or "personalize" (Anya).
func (v *Game) applyItemReward(kind string) (string, error) {
	r := v.quests()
	if r == nil {
		return "", fmt.Errorf("the quest system is not running")
	}

	diff := v.questDifficulty()

	switch kind {
	case "socket":
		if r.rewards.SocketPending <= 0 {
			return "", fmt.Errorf("Larzuk owes no sockets")
		}

		it := v.gameControls.FindItem(func(i *diablo2item.Item) bool {
			_, err := d2reward.LarzukSockets(i.SocketInfo(), diff)
			return err == nil
		})
		if it == nil {
			return "", fmt.Errorf("no item of the hero can be socketed")
		}

		n, _ := d2reward.LarzukSockets(it.SocketInfo(), diff)
		it.SetNumSockets(n)
		r.rewards.SocketPending--
		v.gameControls.SaveItems()
		v.Infof("REWARD socket item=%s sockets=%d difficulty=%d pending=%d", it.CommonCode, n, diff, r.rewards.SocketPending)

		return fmt.Sprintf("%s now has %d sockets", it.CommonCode, n), nil
	case "personalize":
		if r.rewards.PersonalizePending <= 0 {
			return "", fmt.Errorf("Anya owes no personalisation")
		}

		it := v.gameControls.FindItem(func(i *diablo2item.Item) bool {
			return d2reward.CanPersonalize(i.PersonalizeInfo()) == nil
		})
		if it == nil {
			return "", fmt.Errorf("no item of the hero can be personalised")
		}

		it.SetPersonalName(v.localPlayer.Name())
		r.rewards.PersonalizePending--
		v.gameControls.SaveItems()
		v.Infof("REWARD personalize item=%s name=%q pending=%d", it.CommonCode, it.Label(), r.rewards.PersonalizePending)

		return it.Label(), nil
	}

	return "", fmt.Errorf("unknown item reward %q (socket or personalize)", kind)
}

// commandRewardItem is "rewarditem <socket|personalize>".
func (v *Game) commandRewardItem(args []string) error {
	msg, err := v.applyItemReward(args[0])
	if err != nil {
		v.Infof("REWARD refused: %v", err)

		return err
	}

	v.gameControls.Speech.Notice(msg, questNoticeSeconds)

	return nil
}

// questPortal opens the portal a quest asked for (Andariel's corpse portal to
// town, the Duriel's Lair portal, Tyrael's portal to Lut Gholein) when the
// quest knows where it leads.
func (v *Game) questPortal(e d2quest.Effect) {
	if e.Value == 0 {
		v.Infof("QUEST EFFECT portal (%s) [destination unknown, not opened]", e.Note)

		return
	}

	if err := v.commandSpawnPortal([]string{fmt.Sprint(e.Value)}); err != nil {
		v.Infof("QUEST EFFECT portal (%s) to level %d failed: %v", e.Note, e.Value, err)

		return
	}

	v.Infof("QUEST EFFECT portal (%s) opened to level %d", e.Note, e.Value)
}

// ---- act travel ----

// questActChange tells the quest system the hero crossed into the next act by
// an NPC, a portal or by talking, so that the "act finished" words, the
// welcome-back flags and the act unlock are set exactly as the original's
// QUESTS_OnActChangeNpcTravel does.
func (v *Game) questActChange(fromAct, toAct int, via string) {
	r := v.questRT
	if r == nil || toAct != fromAct+1 {
		return
	}

	if !strings.HasPrefix(via, "act:") && via != "portal" {
		return
	}

	var effects []d2quest.Effect

	switch toAct {
	case 2:
		effects = r.g.TravelToAct2()
	case 3:
		effects = r.g.TravelToAct3()
	case 4:
		effects = r.g.TravelToAct4()
	case 5:
		effects = r.g.TravelToAct5()
	default:
		return
	}

	v.Infof("QUEST travel act %d -> %d via=%s", fromAct, toAct, via)
	v.applyQuestEffects(effects)

	r.dirty = true
}

// actPortalLevels are the levels where the original opens the way to the next
// act: the Durance of Hate 3 (Mephisto's red portal) and the Pandemonium
// Fortress (Tyrael's portal after Terror's End, UNVERIFIED).
const (
	levelDurance3 = 102
	levelFortress = 103
)

// maybeOpenActPortal opens the portal to the next act when the hero is where
// it appears and the act's last quest is done; once per visit of the level.
func (v *Game) maybeOpenActPortal() {
	r := v.questRT
	if r == nil || v.localPlayer == nil || v.gameClient == nil || v.levelBusy() {
		return
	}

	level := v.currentLevel()
	if r.actPortalLevel == level {
		return
	}

	var to int

	switch level {
	case levelDurance3:
		to = 4
	case levelFortress:
		to = 5
	default:
		return
	}

	if _, err := d2level.CheckActTravel(d2level.ActOfLevel(level), to, v.heroQuests(), v.heroExpansion(), false); err != nil {
		return
	}

	r.actPortalLevel = level
	dest := d2level.ActStartLevel(to)

	if err := v.commandSpawnPortal([]string{fmt.Sprint(dest)}); err != nil {
		v.Infof("ACT portal to act %d failed: %v", to, err)

		return
	}

	v.Infof("ACT portal act %d -> %d opened in level %d (destination level %d)", d2level.ActOfLevel(level), to, level, dest)
}

// ---- quest drops ----

// questKillDrops gives the quest items a killed monster leaves (the Hellforge
// Hammer, the Mephisto Soulstone) and the Pandemonium keys of the Hell bosses.
func (v *Game) questKillDrops(label string, class int) {
	r := v.questRT
	if r == nil {
		return
	}

	counts := v.gameControls.ItemCountsByCode()
	has := func(code string) bool { return counts[code] > 0 || r.g.Items[code] > 0 }

	done := false

	if q := r.g.Quest(d2quest.QuestHellforge); q != nil {
		done = r.g.Rec.Get(q.Slot, d2quest.FlagRewardGranted) || r.g.Rec.Get(q.Slot, d2quest.FlagPrimaryGoal)
	}

	for _, d := range d2reward.DropsFor(label, has, done) {
		v.Infof("QUEST DROP %s from %q (%s)", d.Code, label, d.Why)
		v.spawnQuestItem(d.Code)
	}

	v.uberKeyDrop(class)
}

// ---- Horadric Cube ----

// transmuteCube is the Transmute button for the recipes of the quests and the
// Pandemonium event: Khalim's Will, the Horadric Staff, the three keys and the
// three organs. Other recipes are not in the engine.
func (v *Game) transmuteCube() (string, error) {
	codes := v.gameControls.CubeCodes()

	rec, ok := d2cube.Match(codes, v.heroExpansion())
	if !ok {
		v.Infof("CUBE no quest recipe for %v", codes)

		return "", fmt.Errorf("the cube holds no recipe (%v)", codes)
	}

	n := v.gameControls.CubeClear()
	v.Infof("CUBE transmute %q consumed=%d", rec.Name, n)

	if rec.Kind == d2cube.ResultItem {
		if err := v.gameControls.GiveItemToCube(rec.Code); err != nil {
			v.Infof("CUBE result %s goes to the ground: %v", rec.Code, err)
			v.spawnQuestItem(rec.Code)
		} else {
			v.Infof("CUBE result %s is in the cube", rec.Code)
		}

		v.questItemPickedUp(rec.Code) // the quest system counts the new Staff / Will

		return rec.Code, nil
	}

	acts, mine := v.uberCube(rec)
	if !mine {
		return "", fmt.Errorf("recipe %q is not handled", rec.Name)
	}

	return fmt.Sprintf("%d actions", acts), nil
}

// commandTransmute is "transmute".
func (v *Game) commandTransmute(_ []string) error {
	_, err := v.transmuteCube()

	return err
}

// giveRewardTestItem gives the hero an item of a given quality (the autotest).
func (v *Game) giveRewardTestItem(code string, quality int) (string, error) {
	return v.gameControls.GiveItemQuality(code, quality, 40)
}
