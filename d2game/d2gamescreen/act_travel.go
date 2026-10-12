package d2gamescreen

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// Travel between acts. The rules (which NPC or portal, which quest) live in
// d2level/acttravel.go with their verified/unverified status; this file is the
// glue: menu rows, the console commands the autotest uses, the level change
// with its fade (an act change is a level change to the start level of the
// destination act, SERVER_ChangePlayerLevel -> SERVER_ChangePlayerAct) and the
// log lines the autotest reads.

type travelState struct {
	free        bool // debug: skip the quest and NPC rules
	savedDone   bool // the hero was moved to the act he was saved in
	moviePlayed bool // the player's play flag (PlayerData+0x4c): an act movie was already sent this session
}

func (v *Game) currentAct() int { return d2level.ActOfLevel(v.currentLevel()) }

func (v *Game) heroQuests() *d2s.QuestRecord {
	if v.gameClient.Progress == nil {
		return nil
	}

	return v.gameClient.Progress.QuestRecord(int(v.gameClient.Difficulty))
}

// heroExpansion: the Lord of Destruction rule is only enforced for heroes
// imported from a .d2s (heroes created in the engine do not record the
// choice, and the engine always has the expansion data).
func (v *Game) heroExpansion() bool {
	c := v.gameClient

	return !c.FromSave || c.Expansion
}

// travelToAct checks the rule for the trip and starts the level change to the
// town of the destination act.
func (v *Game) travelToAct(to int, via string) error {
	from := v.currentAct()

	rule, err := d2level.CheckActTravel(from, to, v.heroQuests(), v.heroExpansion(), v.travel.free)
	if err != nil {
		v.Infof("TRAVEL refused act %d -> %d via=%s: %v", from, to, via, err)
		return err
	}

	if v.levelBusy() {
		return fmt.Errorf("a level change is already running")
	}

	// the act movie (if the exe sends one) plays first, over the standing game; the trip starts when it is over
	if m, ok := v.actMovie(from, to, via); ok && v.playGameMovies(m.Movie, m.Then) {
		v.movie.after = func() {
			if err := v.beginActTravel(from, to, via, rule); err != nil {
				v.Errorf("TRAVEL after the movie: %v", err)
			}
		}

		return nil
	}

	return v.beginActTravel(from, to, via, rule)
}

// beginActTravel starts the level change to the town of the destination act and marks the trip.
func (v *Game) beginActTravel(from, to int, via string, rule d2level.ActRule) error {
	dest := d2level.ActStartLevel(to)
	forward := to > from

	v.Infof("TRAVEL act %d -> %d via=%s level=%d (%s) rule: quest slot %d, expansion=%v, rule verified=%v, free=%v",
		from, to, via, dest, v.levelName(dest), rule.QuestSlot, rule.NeedExpansion, rule.Verified, v.travel.free)

	if !v.startLevelChange(dest, d2level.StartActChange, "act:"+via) {
		return fmt.Errorf("the engine cannot load level %d", dest)
	}

	// the server marks the act just left as finished for a forward trip made
	// by the travel NPC, by talking, and by the Mephisto portal (VERIFIED:
	// QUESTS_OnHellPortalActChange sets slot 23; earlier code excluded it)
	v.levels.trans.actFinished = forward && (via == "npc" || via == "talk" || via == "portal")

	// the three travel rows of the NPCs (Warriv, Meshif, Tyrael) open the waypoint of the town they end in when the menu
	// action runs (VERIFIED, TRADE_ServerHandleNpcMenuAction 0x577b70); the Mephisto portal does not
	if forward && (via == "npc" || via == "talk") {
		v.activateTravelWaypoint(dest)
	}

	return nil
}

// withTravelRows adds the east-bound row of Warriv (Act 1) and Meshif (Act 2)
// when the rule allows the trip. The back-bound rows are in the static table.
func (v *Game) withTravelRows(class int, rows []d2player.NPCMenuRow) []d2player.NPCMenuRow {
	rule, ok := d2level.RuleForNPC(class)
	if !ok || rule.To < rule.From || (rule.Via != d2level.ViaNPC && !(rule.Via == d2level.ViaTalk && rule.Row != "")) {
		return rows
	}

	if _, err := d2level.CheckActTravel(rule.From, rule.To, v.heroQuests(), v.heroExpansion(), v.travel.free); err != nil {
		return rows
	}

	row := d2player.RowGoEast

	switch class {
	case d2level.ClassMeshif1:
		row = d2player.RowSailEast
	case d2level.ClassTyrael2:
		row = d2player.RowTravelHarrogath
	}

	return append(append([]d2player.NPCMenuRow{}, rows...), row)
}

// travelFromNPC runs a travel row of an NPC menu.
func (v *Game) travelFromNPC(npc d2interface.MapEntity, row d2player.NPCMenuRow) {
	rule, ok := d2level.RuleForNPC(v.npcClassID(npc))
	if !ok {
		v.Infof("NPC menu: %s: %q has no travel rule", row.Action, npc.Label())
		return
	}

	v.gameControls.NPCMenu.Close()
	v.npcTarget = nil

	via := "npc"
	if rule.Via == d2level.ViaTalk {
		via = "talk" // Tyrael: the cinematic rule keys on it
	}

	_ = v.travelToAct(rule.To, via)
}

// commandNPCTravel is "npctravel <class>": the server NPC action no client menu sends (Cain in Harrogath, class 520:
// level 0x67 without a quest test, VERIFIED 0x577b70).
func (v *Game) commandNPCTravel(args []string) error {
	class, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid class %q", args[0])
	}

	rule, ok := d2level.ServerTravel(class)
	if !ok {
		return fmt.Errorf("class %d has no server travel action", class)
	}

	if v.levelBusy() {
		return fmt.Errorf("a level change is already running")
	}

	if from := v.currentAct(); from != rule.From {
		return fmt.Errorf("the action is for act %d (hero in act %d)", rule.From, from)
	}

	return v.beginActTravel(rule.From, rule.To, "server", rule)
}

// logActArrival logs what the new act's town offers: NPCs, waypoint and stash
// objects (AUTOSCRIPT scenarios read these lines).
func (v *Game) logActArrival(level int) {
	wp, stash := 0, 0

	for _, e := range v.gameClient.MapEngine.Entities() {
		ob, ok := e.(*d2mapentity.Object)
		if !ok {
			continue
		}

		switch {
		case ob.Kind() == d2level.ObjectWaypoint:
			wp++
		case strings.EqualFold(ob.Record().Name, "Bank"):
			stash++
		}
	}

	v.Infof("ACT arrival level=%d act=%d (%s) waypoints=%d stash=%d", level, d2level.ActOfLevel(level), v.levelName(level), wp, stash)
}

// advanceSavedAct moves a hero saved in act 2-5 to the town of that act once
// the game is running (the server places every hero in the Rogue Encampment).
func (v *Game) advanceSavedAct() {
	c := v.gameClient
	if v.travel.savedDone || v.localPlayer == nil || v.gameControls == nil || c.SavedAct < 2 || c.SavedAct > 5 {
		return
	}

	v.travel.savedDone = true

	if os.Getenv("OD2_AUTOLEVEL") != "" || os.Getenv("OD2_AUTOMAP") != "" {
		return
	}

	dest := d2level.ActStartLevel(c.SavedAct)
	v.Infof("ACT load: the hero was saved in act %d, entering level %d", c.SavedAct, dest)
	v.startLevelChange(dest, d2level.StartActChange, "load")
}

// commandCompleteQuest is "completequest <act> <quest>": marks the quest done.
func (v *Game) commandCompleteQuest(args []string) error {
	act, err1 := strconv.Atoi(args[0])
	quest, err2 := strconv.Atoi(args[1])

	slot, ok := d2s.QuestSlot(act, quest)
	if err1 != nil || err2 != nil || !ok {
		return fmt.Errorf("invalid quest %s %s", args[0], args[1])
	}

	q := v.heroQuests()
	if q == nil {
		return fmt.Errorf("the hero has no quest record")
	}

	q.Set(slot, d2s.QuestBitDone)
	v.Infof("QUEST act=%d quest=%d slot=%d marked done", act, quest, slot)

	// the Act 5 quests open one after the other: a running quest system must see the chain move on
	if chain := []int{d2quest.QuestSiege, d2quest.QuestRescue, d2quest.QuestPrison, d2quest.QuestBetrayal, d2quest.QuestRite}; act == 5 &&
		quest >= 1 && quest <= len(chain) && v.questRT != nil {
		v.questRT.g.ForceDone(chain[quest-1])
	}

	return nil
}

// commandResetQuests is "resetquests": clears the quest record of the current
// difficulty (debug; local copy only, so the rules can be tested from zero).
func (v *Game) commandResetQuests(_ []string) error {
	q := v.heroQuests()
	if q == nil {
		return fmt.Errorf("the hero has no quest record")
	}

	for slot := 0; slot < d2s.QuestSlots; slot++ {
		q.SetSlot(slot, 0)
	}

	v.Infof("QUEST record cleared")
	v.restartQuests()

	// the quest system holds state of its own (quest states, heard lines): build it again from the cleared record
	if v.questRT != nil {
		v.questRT = nil
		v.quests()
	}

	return nil
}

// commandTravelFree is "travelfree <0|1>".
func (v *Game) commandTravelFree(args []string) error {
	v.travel.free = args[0] != "0"
	v.Infof("TRAVEL free mode %v", v.travel.free)

	return nil
}

// commandTravel is "travel <act>".
func (v *Game) commandTravel(args []string) error {
	act, err := strconv.Atoi(args[0])
	if err != nil || act < 1 || act > d2level.NumActs {
		return fmt.Errorf("invalid act %q", args[0])
	}

	return v.travelToAct(act, "console")
}

// Travel implements d2autoscript.TravelHost: the trip as if the hero used the
// travel NPC or portal of his act ("npc" for the forward NPC trips, "portal"
// for Act 3 -> 4, "talk" for Act 4 -> 5), with the rule checks.
func (h autoScriptHost) Travel(act int) error {
	from := h.v.currentAct()
	via := "npc"

	if r, ok := d2level.RuleFor(from, act); ok {
		switch r.Via {
		case d2level.ViaPortal:
			via = "portal"
		case d2level.ViaTalk:
			via = "talk"
		}
	}

	return h.v.travelToAct(act, via)
}

// activateTravelWaypoint activates the waypoint of the town an NPC trip ends in (d2level.NpcWaypoints).
func (v *Game) activateTravelWaypoint(town int) {
	for _, n := range d2level.NpcWaypoints {
		if n.Level != town {
			continue
		}

		changed, err := v.gameClient.SetWaypoint(town, true)
		v.Infof("WAYPOINT travel activated level=%d (%s) changed=%v err=%v", town, v.levelName(town), changed, err)

		return
	}
}
