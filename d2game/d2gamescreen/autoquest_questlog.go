package d2gamescreen

import (
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
)

// OD2_AUTOQUEST=questlog opens the quest log for several quest states and logs the text of each (AUTOQUEST questlog
// lines: act, quest, icon class, page, string key, text). The pages and keys are those of the client table of Game.exe
// (d2quest.ClientLogView); every state is checked against the key the original shows.

func init() {
	actStageOrder["questlog"] = []string{"questlog"}
	actStageSlots["questlog"] = [2][]int{nil, nil} // a clean Acts 2-5 record
	autoQuestNames["questlog"] = "questlog"
}

// logShows opens the log on the quest and checks the key and the icon class the client rules give.
func (a *autoQuest) logShows(h autoHost, why string, act, index int, wantKey string, wantCat d2quest.LogCategory) {
	h.OpenLog(act, index)

	for _, l := range h.Q().Log() {
		if l.Act != act || l.Index != index {
			continue
		}

		h.Infof("AUTOQUEST questlog [%s] act=%d quest=%d cat=%d page=%d key=%s text=%q", why, act, index, l.View.Cat, l.View.Page,
			l.View.Key, shorten(h.LogText(act, index), 70))
		a.expect(h, why+": "+wantKey, l.View.Key == wantKey && l.View.Cat == wantCat)

		return
	}

	a.expect(h, why+": quest in the log", false)
}

func (a *autoQuest) stageQuestLog() {
	const (
		a1Den, a1Tools         = 1, 3
		a2Staff                = 2
		a3Bird                 = 4
		a5Siege                = 1
		active, locked, animat = d2quest.LogCatActive, d2quest.LogCatLocked, d2quest.LogCatAnimating
	)

	a.once("questlog: nothing started", func(h autoHost, a *autoQuest) {
		a.logShows(h, "fresh Den", 1, a1Den, "", locked)
		a.logShows(h, "fresh Cain", 1, 4, "", locked)
		a.logShows(h, "fresh Staff", 2, a2Staff, "", locked)
	})
	a.once("questlog: Den of Evil", func(h autoHost, a *autoQuest) {
		a.talk(h, d2quest.NPCAkara)
		a.logShows(h, "Den: Akara sent the hero", 1, a1Den, "qstsa1q11", active)
		a.move(h, 2)
		a.move(h, d2quest.LevelDenOfEvil)
		h.Q().SetDenMonsters(6)
		a.logShows(h, "Den: entered the den", 1, a1Den, "qstsa1q12", active)

		for i := 0; i < 6; i++ {
			a.kill(h, d2quest.Event{Monster: 3})
		}

		a.tick(h, 8)
		a.logShows(h, "Den: cleared, reward pending", 1, a1Den, "qstsa1q15", active)
		a.move(h, 2)
		a.move(h, d2quest.LevelRogueEncampment)
		a.talk(h, d2quest.NPCAkara)
		a.logShows(h, "Den: reward taken, not looked at", 1, a1Den, "qstsComplete", animat)
	})
	a.once("questlog: Tools of the Trade", func(h autoHost, a *autoQuest) {
		a.pickup(h, d2quest.ItemHoradricMalus)
		a.logShows(h, "Tools: malus in the pack", 1, a1Tools, "qstsa1q32", active)
	})
	a.once("questlog: Horadric Staff", func(h autoHost, a *autoQuest) {
		a.pickup(h, d2quest.ItemHoradricScroll)
		a.logShows(h, "Staff: scroll in the pack", 2, a2Staff, "qstsa2q21", active)
	})
	a.once("questlog: Golden Bird", func(h autoHost, a *autoQuest) {
		a.pickup(h, d2quest.ItemJadeFigurine)
		a.logShows(h, "Bird: figurine in the pack", 3, a3Bird, "qstsa3q41", active)
	})
	a.once("questlog: Siege on Harrogath", func(h autoHost, a *autoQuest) {
		a.move(h, d2quest.LevelHarrogath)
		a.says(h, "Larzuk offers the quest", d2quest.NPCLarzuk, 20077)
		a.logShows(h, "Siege: Larzuk sent the hero", 5, a5Siege, "qsta5q11", active)
		a.move(h, d2quest.LevelBloodyFoothills)
		a.kill(h, d2quest.Event{Monster: 999, Super: "Shenk the Overseer"})
		a.move(h, d2quest.LevelHarrogath)
		a.logShows(h, "Siege: Shenk dead", 5, a5Siege, "qsta5q13", active)
		a.says(h, "Larzuk's thanks", d2quest.NPCLarzuk, 20090)
		a.logShows(h, "Siege: Larzuk will add sockets", 5, a5Siege, "qsta5q14", active)
		h.Apply(h.Q().ClaimSocket())
		a.logShows(h, "Siege: sockets added", 5, a5Siege, "qstsComplete", animat)
	})
}
