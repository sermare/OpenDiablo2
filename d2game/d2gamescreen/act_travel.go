package d2gamescreen

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2level"
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
	free      bool // debug: skip the quest and NPC rules
	savedDone bool // the hero was moved to the act he was saved in
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

	dest := d2level.ActStartLevel(to)
	forward := to > from

	v.Infof("TRAVEL act %d -> %d via=%s level=%d (%s) rule: quest slot %d, expansion=%v, rule verified=%v, free=%v",
		from, to, via, dest, v.levelName(dest), rule.QuestSlot, rule.NeedExpansion, rule.Verified, v.travel.free)

	if !v.startLevelChange(dest, d2level.StartActChange, "act:"+via) {
		return fmt.Errorf("the engine cannot load level %d", dest)
	}

	// the server marks the act just left as finished for a forward trip made
	// by the travel NPC or by talking (not for the Mephisto portal)
	v.levels.trans.actFinished = forward && (via == "npc" || via == "talk")

	return nil
}

// withTravelRows adds the east-bound row of Warriv (Act 1) and Meshif (Act 2)
// when the rule allows the trip. The back-bound rows are in the static table.
func (v *Game) withTravelRows(class int, rows []d2player.NPCMenuRow) []d2player.NPCMenuRow {
	rule, ok := d2level.RuleForNPC(class)
	if !ok || rule.Via != d2level.ViaNPC || rule.To < rule.From {
		return rows
	}

	if _, err := d2level.CheckActTravel(rule.From, rule.To, v.heroQuests(), v.heroExpansion(), v.travel.free); err != nil {
		return rows
	}

	row := d2player.RowGoEast
	if class == d2level.ClassMeshif1 {
		row = d2player.RowSailEast
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

	_ = v.travelToAct(rule.To, "npc")
}

// travelOnTalk is the effect of talking to Tyrael in the Pandemonium Fortress
// once Diablo is dead (UNVERIFIED mechanism, see d2level/acttravel.go).
func (v *Game) travelOnTalk(npc d2interface.MapEntity) {
	rule, ok := d2level.RuleForNPC(v.npcClassID(npc))
	if !ok || rule.Via != d2level.ViaTalk {
		return
	}

	v.gameControls.NPCMenu.Close()
	v.npcTarget = nil

	_ = v.travelToAct(rule.To, "talk")
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
