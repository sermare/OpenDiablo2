package d2quest

import "testing"

func speaks(g *Game, npc, msg int, mode Mode) bool {
	for _, l := range g.Activate(npc).Lines {
		if l.Msg == msg && l.Mode == mode {
			return true
		}
	}

	return false
}

// TestBossKillSpeech pins boss_speech.go (OnNpcInteract 0x5b9e20 / 0x5b23b0 / 0x58b7d0 and the message-heard handlers):
// the NPC lines after the kills are spoken when the hero talks to the NPC, once, then they become topics.
func TestBossKillSpeech(t *testing.T) {
	for _, c := range []struct {
		name      string
		classic   bool
		killLevel int
		kill      Event
		town      int
		npc, msg  int
		otherNPC  int
		otherMsg  int
		repeats   bool // the line is spoken at every talk (Tyrael's end line has no heard bit)
		topic     bool // after hearing it the NPC offers a topic
	}{
		{name: "mephisto/alkor", kill: Event{Monster: NPCMephisto, Name: "Mephisto"}, town: LevelKurastDocktown,
			npc: NPCAlkor, msg: 657, otherNPC: NPCOrmus, otherMsg: 658, topic: true},
		{name: "diablo expansion/tyrael", kill: Event{Monster: NPCDiablo, Name: "Diablo"}, town: LevelPandemonium,
			npc: NPCTyrael2, msg: 20000, otherNPC: NPCCain4, otherMsg: 20001},
		{name: "diablo classic/tyrael", classic: true, kill: Event{Monster: NPCDiablo, Name: "Diablo"}, town: LevelPandemonium,
			npc: NPCTyrael2, msg: 684, otherNPC: NPCCain4, otherMsg: 685},
		{name: "baal/tyrael", killLevel: LevelWorldstoneChamber, kill: Event{Monster: NPCBaalCrab, Name: "Baal"}, town: LevelHarrogath,
			npc: NPCTyrael3, msg: 20175, otherNPC: NPCLarzuk, otherMsg: 20178, repeats: true},
	} {
		g, _ := newGame(t)
		g.Expansion = !c.classic
		g.Level = c.town

		if speaks(g, c.npc, c.msg, ModeSpoken) {
			t.Errorf("%s: speaks before the kill", c.name)
		}

		ev := c.kill
		ev.Kind, ev.Level = EvMonsterKilled, c.killLevel

		if ev.Level == 0 {
			ev.Level = c.town
		}

		g.Level = ev.Level
		g.Dispatch(ev)
		g.Level = c.town

		if !speaks(g, c.npc, c.msg, ModeSpoken) || !speaks(g, c.otherNPC, c.otherMsg, ModeSpoken) {
			t.Fatalf("%s: lines not spoken after the kill", c.name)
		}

		g.Hear(c.npc, c.msg)
		g.Hear(c.otherNPC, c.otherMsg)

		if again := speaks(g, c.npc, c.msg, ModeSpoken); again != c.repeats {
			t.Errorf("%s: spoken again = %v after hearing, want %v", c.name, again, c.repeats)
		}

		if speaks(g, c.otherNPC, c.otherMsg, ModeSpoken) {
			t.Errorf("%s: second NPC speaks again", c.name)
		}

		if c.topic && !speaks(g, c.otherNPC, c.otherMsg, ModeTopic) {
			t.Errorf("%s: no topic for the second NPC after hearing", c.name)
		}

		legacy, _ := newGame(t)
		legacy.Expansion, legacy.LegacyBossBits, legacy.Level = !c.classic, true, c.town

		if speaks(legacy, c.npc, c.msg, ModeSpoken) {
			t.Errorf("%s: speaks in legacy mode before the kill", c.name)
		}
	}
}
