package d2gamescreen

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// questNPC names the NPC of a conversation. ent is nil when the NPC is not on
// the map (the autotest talks to NPCs by class while the hero stands in a cave).
type questNPC struct {
	class int
	label string
	ent   d2interface.MapEntity
}

func (v *Game) npcRef(e d2interface.MapEntity) questNPC {
	return questNPC{class: v.npcClassID(e), label: e.Label(), ent: e}
}

// findNPC returns the map entity of an NPC class, if it is on the map.
func (v *Game) findNPC(class int) (d2interface.MapEntity, bool) {
	for _, e := range v.gameClient.MapEngine.Entities() {
		if v.npcClassID(e) == class {
			return e, true
		}
	}

	return nil, false
}

// refForClass builds a reference for an NPC by class, using the map entity if present.
func (v *Game) refForClass(class int) questNPC {
	if e, ok := v.findNPC(class); ok {
		return v.npcRef(e)
	}

	name := d2player.NPCClassName(class)
	if name == "" {
		name = fmt.Sprintf("npc%d", class)
	}

	return questNPC{class: class, label: name}
}

// speechText strips the leading number the speech entries of string.tbl carry
// ("43 There is a place..."; its meaning - probably a length - is UNVERIFIED).
func speechText(s string) string {
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}

	if i > 0 && i < len(s) && (s[i] == ' ' || s[i] == '\n' || s[i] == '\t' || s[i] == '\r') {
		return strings.TrimSpace(s[i:])
	}

	return s
}

func speechSeconds(text string) float64 {
	return speechSecondsBase + float64(len(text))/speechCharsPerSec
}

// questTalk is the Talk row for an NPC on the map.
func (v *Game) questTalk(npc d2interface.MapEntity) bool {
	return v.questTalkRef(v.npcRef(npc))
}

// questClose is the player leaving an NPC dialog (quest event 2).
func (v *Game) questClose(npc d2interface.MapEntity) {
	v.questCloseRef(v.npcRef(npc))
}

// questTopic plays the topic the player picked in the Talk submenu.
func (v *Game) questTopic(npc d2interface.MapEntity, msg int) {
	v.questTopicRef(v.npcRef(npc), msg)
}

// questTalkRef: the quest system builds the speech list for the NPC (ev0), the
// next unheard spoken line is voiced, and when nothing is left to say the mode
// 2 topics are offered. It reports whether the quest system handled the click
// (otherwise the plain greeting plays).
func (v *Game) questTalkRef(npc questNPC) bool {
	r := v.quests()
	if r == nil {
		return false
	}

	r.g.Hero.Level = v.localPlayer.Stats.Level
	d := r.g.Activate(npc.class)

	if s, ok := d.NextSpoken(r.g); ok {
		v.questSpeak(npc, s)

		return true
	}

	if topics := d.Topics(); len(topics) > 0 {
		v.questTopicMenu(npc, topics)

		return true
	}

	// nothing to say: the dialog closes (ev2) and the plain greeting plays
	v.questCloseRef(npc)

	return false
}

// questSpeak voices one message: resolves the Sounds.txt row of the message id
// and the text from the string table, plays and shows it, then reports it as
// heard (the client acknowledgement that drives the quest state).
func (v *Game) questSpeak(npc questNPC, s d2quest.Speech) {
	r := v.quests()

	snd, hasSound := d2quest.SoundForMessage(s.Msg)
	key := d2quest.TextKey(s.Msg)

	text := ""
	if key != "" {
		if t := v.asset.TranslateString(key); t != "" && t != key {
			text = speechText(t)
		}
	}

	v.Infof("QUEST SPEECH npc=%q class=%d msg=%d mode=%d quest=%d sound=%d handle=%s file=%s key=%s text=%q",
		npc.label, npc.class, s.Msg, s.Mode, s.Quest, snd.Index, snd.Handle, snd.File, key, shorten(text, 60))

	r.spoken = append(r.spoken, s.Msg)

	if hasSound && os.Getenv("OD2_AUTOTEST_MUTE") == "" {
		if v.soundEngine.PlaySoundID(snd.Index) == nil {
			v.Warningf("could not play message %d (sound %d)", s.Msg, snd.Index)
		}
	}

	if text != "" {
		v.gameControls.Speech.Show(npc.label, text, speechSeconds(text))
	}

	v.applyQuestEffects(r.g.Hear(npc.class, s.Msg))

	r.dirty = true

	// the NPC finished speaking: when nothing spoken is left the dialog closes
	if _, more := r.g.Activate(npc.class).NextSpoken(r.g); !more {
		v.questCloseRef(npc)
	}
}

func (v *Game) questCloseRef(npc questNPC) {
	r := v.quests()
	if r == nil {
		return
	}

	v.applyQuestEffects(r.g.Close(npc.class))

	r.dirty = true
}

// questTopicMenu lists the topics of the Talk submenu. Without a map entity to
// anchor the menu the topics are logged only.
func (v *Game) questTopicMenu(npc questNPC, topics []d2quest.Speech) {
	r := v.quests()
	rows := make([]d2player.NPCMenuRow, 0, len(topics))

	for _, t := range topics {
		rows = append(rows, d2player.NPCMenuRow{StringID: t.Msg, Fallback: v.topicLabel(r, t), Action: d2player.NPCActionTopic})
	}

	labels := make([]string, 0, len(rows))
	for _, row := range rows {
		labels = append(labels, fmt.Sprintf("%d:%s", row.StringID, row.Fallback))
	}

	v.Infof("QUEST TOPICS npc=%q class=%d topics=%v", npc.label, npc.class, labels)

	r.topics = topics

	if npc.ent == nil {
		return
	}

	menu := v.gameControls.NPCMenu
	menu.Open(npc.label, rows, 0, 0, func(row d2player.NPCMenuRow) {
		v.onNPCMenuChoice(npc.ent, row)
	})
	v.anchorNPCMenu(menu, npc.ent)
}

// topicLabel names a topic by its quest (UNVERIFIED: the real client's topic
// captions were not decoded; the quest title is the best fit).
func (v *Game) topicLabel(r *questRuntime, t d2quest.Speech) string {
	if q := r.g.Quest(t.Quest); q != nil && q.LogIndex > 0 {
		key := fmt.Sprintf("qstsa%dq%d", q.Act+1, q.LogIndex)
		if title := v.asset.TranslateString(key); title != "" && title != key {
			return title
		}
	}

	return fmt.Sprintf("Topic %d", t.Msg)
}

// questTopicRef plays the topic the player picked.
func (v *Game) questTopicRef(npc questNPC, msg int) {
	r := v.quests()
	if r == nil {
		return
	}

	for _, l := range r.g.Activate(npc.class).Lines {
		if l.Msg == msg {
			v.gameControls.NPCMenu.Close()
			v.questSpeak(npc, l)

			return
		}
	}
}

func shorten(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		return s[:n] + "..."
	}

	return s
}

// armReturnGreeting arms the NPC's welcome-back greeting when the quest system
// says it is pending, and clears the flag (client packet 0x4d).
func (v *Game) armReturnGreeting(name string) {
	r := v.questRT
	if r == nil {
		return
	}

	for class := 100; class < 530; class++ {
		if n := d2player.NPCClassName(class); n != "" && strings.EqualFold(n, name) && r.g.ReturnGreetingPending(class) {
			v.returnGreet.Arm(strings.ToLower(name))
			r.g.ClearReturnGreeting(class)
			v.Infof("QUEST welcome-back greeting armed for %s", name)

			return
		}
	}
}
