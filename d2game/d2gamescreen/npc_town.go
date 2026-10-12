package d2gamescreen

import (
	"fmt"
	"os"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2npc"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// The town NPC rules of Game.exe (d2common/d2npc) joined to the running game: the menu rows of an NPC, the
// healing of the six healers, the once-per-room greeting, and the Talk submenu with its "introduction" and
// "gossip" lines.

// npcRoomTiles is the edge of the hero's "room" in tiles for the room change that re-allows the greeting
// (PLAYER_UpdateRoomChangeState 0x45c640 fires on a Room1 change; the town levels are DS1 presets that the
// exe cuts into 5x5 tile rooms, UNVERIFIED).
const npcRoomTiles = 5

// npcTownState is the client NPC memory (the state array at 0x725968) of the running game.
type npcTownState struct {
	gate     *d2npc.Gate
	seed     d2npc.Seed
	room     [3]int
	haveRoom bool
}

func (v *Game) npcTown() *npcTownState {
	if v.npcTownRT == nil {
		s := &npcTownState{gate: d2npc.NewGate()}

		// the exe rolls with the hero unit's own seed; this one comes from the map seed so a run is repeatable
		if v.gameClient != nil && v.gameClient.MapEngine != nil {
			s.seed.V = uint64(v.gameClient.MapEngine.Seed())<<32 | 0x9e3779b9
		} else {
			s.seed.V = 0x9e3779b9
		}

		v.npcTownRT = s
	}

	return v.npcTownRT
}

// heroExeClass is the hero class in the exe's numbering (0 amazon, 1 sorceress, 2 necromancer, 3 paladin,
// 4 barbarian, 5 druid, 6 assassin), -1 without a hero.
func (v *Game) heroExeClass() int {
	if v.localPlayer == nil {
		return -1
	}

	switch v.localPlayer.Class {
	case d2enum.HeroAmazon:
		return 0
	case d2enum.HeroSorceress:
		return 1
	case d2enum.HeroNecromancer:
		return 2
	case d2enum.HeroPaladin:
		return 3
	case d2enum.HeroBarbarian:
		return 4
	case d2enum.HeroDruid:
		return 5
	case d2enum.HeroAssassin:
		return 6
	default:
		return -1
	}
}

// npcQuestRecord is the hero's quest record: the live one of the quest system when it runs.
func (v *Game) npcQuestRecord() *d2s.QuestRecord {
	if r := v.questRT; r != nil && r.g != nil && r.g.Rec != nil {
		return r.g.Rec
	}

	return v.heroQuests()
}

// npcQuestFlag reads a bit of the hero's quest record.
func (v *Game) npcQuestFlag(quest, bit int) bool {
	if q := v.npcQuestRecord(); q != nil {
		return q.Get(quest, bit)
	}

	return false
}

// npcCtx collects what the menu rules read from the game.
func (v *Game) npcCtx() d2npc.Ctx {
	c := d2npc.Ctx{Expansion: v.heroExpansion(), Quest: v.npcQuestFlag,
		RespecPanelFlag: d2npc.RespecPanelFlagFor(v.questDifficulty())}

	if v.localPlayer != nil && v.localPlayer.Stats != nil {
		c.HeroLevel = v.localPlayer.Stats.Level
	}

	if info, ok := v.deadMerc(); ok {
		c.MercDead, c.MercCost = true, info.ReviveCost
	}

	if v.gameControls != nil && v.gameControls.Identify != nil {
		c.Unidentified = len(v.gameControls.Identify.Unidentified())
	}

	return c
}

// npcMenuRows turns the exe's rows for the NPC class into menu rows. known is false for a class without a
// row (the NPC then only talks). Qual-Kehk gets the Hire row that the static table does not give him
// (UNVERIFIED how the original reaches his hire list: no code in the analysis adds the row).
func (v *Game) npcMenuRows(class int) (rows []d2player.NPCMenuRow, known bool) {
	opts, known := d2npc.Options(class, v.npcCtx())
	if !known {
		rows, _ = d2player.NPCMenuFor(class)

		return rows, false
	}

	if class == d2npc.ClassQualKehk {
		opts = append([]d2npc.Option{opts[0], {StringID: d2npc.StrHire, Kind: d2npc.KindHire}}, opts[1:]...)
	}

	for _, o := range opts {
		rows = append(rows, v.npcRowFor(o))
	}

	return rows, true
}

func (v *Game) npcRowFor(o d2npc.Option) d2player.NPCMenuRow {
	switch o.Kind {
	case d2npc.KindRevive:
		name := "mercenary"
		if info, ok := v.deadMerc(); ok {
			name = v.mercDisplayName(info.Rec, int(info.Save.NameID))
		}

		return d2player.NPCMenuRow{StringID: o.StringID, Fallback: fmt.Sprintf("Resurrect %s: %d", name, o.Cost),
			Action: d2player.NPCActionReviveMerc}
	case d2npc.KindIdentify:
		row, _ := d2player.NPCRowFor(d2npc.StrIdentify)
		if o.Cost > 0 {
			label := v.asset.TranslateString("NPCIdentify2")
			if label == "" || label == "NPCIdentify2" {
				label = "Identify Items: "
			}

			row.Key, row.StringID, row.Fallback = "", d2npc.StrIdentifyFee, fmt.Sprintf("%s%d", label, o.Cost)
		}

		return row
	default:
		row, ok := d2player.NPCRowFor(o.StringID)
		if !ok {
			row = d2player.NPCMenuRow{StringID: o.StringID, Fallback: fmt.Sprintf("row %#x", o.StringID), Action: d2player.NPCActionTalk}
		}

		return row
	}
}

// watchNPCRoom runs every frame: when the hero steps into another room every greeting is allowed again and
// the NPCs that spoke get a new gossip line (UI_ResetNpcMenuEntriesOnPlayerChange 0x4b0630). The exe also
// cancels an open NPC interaction there; OpenDiablo2 closes the menu by distance instead.
func (v *Game) watchNPCRoom() {
	if v.localPlayer == nil {
		return
	}

	x, y := v.localPlayer.GetPositionF()
	key := [3]int{v.currentAct(), int(x) / npcRoomTiles, int(y) / npcRoomTiles}

	s := v.npcTown()
	if s.haveRoom && key == s.room {
		return
	}

	first := !s.haveRoom
	s.room, s.haveRoom = key, true

	if !first {
		s.gate.RoomChanged(v.rerollGossip)
	}
}

func (v *Game) rerollGossip(e *d2npc.Entry) int {
	s := v.npcTown()

	return d2npc.PickGossip(e, v.heroExeClass(), v.npcQuestFlag, s.seed.Below)
}

// greetOnOpen is the greeting step of UI_StartNpcInteraction: it plays the NPC's spoken greeting when the
// menu opens, once per room (and the return greeting when the quest system armed it).
func (v *Game) greetOnOpen(npc d2interface.MapEntity) {
	class := v.npcClassID(npc)
	name := npc.Label()
	base := strings.ToLower(strings.TrimPrefix(name, "Deckard "))

	// the quest system arms the return line (client packet 0x4d); returnGreet keeps it for the picker
	v.armReturnGreeting(base)

	play, _ := v.npcTown().gate.OpenGreeting(class)
	if !play {
		// a pending return flag is spent even when no greeting plays
		if v.returnGreet != nil {
			v.returnGreet.Take(base)
		}

		v.Infof("NPC greeting skipped: %q already greeted in this room", name)

		return
	}

	v.playNPCGreeting(name)
}

// healOnTalk is NPCSRV_RestoreVitalsOnTalk: Akara, Malah, Atma, Fara, Jamella and Ormus fill the hero's life,
// mana and stamina when the interaction starts and cure it of poison, freeze and the curable states.
func (v *Game) healOnTalk(npc d2interface.MapEntity) {
	class := v.npcClassID(npc)
	p := v.localPlayer

	if !d2npc.IsHealer(class) || p == nil || p.Stats == nil {
		return
	}

	st := p.Stats
	before := d2npc.Vitals{Life: st.Health, MaxLife: st.MaxHealth, Mana: st.Mana, MaxMana: st.MaxMana,
		Stamina: st.Stamina, MaxStamina: float64(st.MaxStamina)}
	after, changed := d2npc.Restore(before)
	st.Health, st.Mana, st.Stamina = after.Life, after.Mana, after.Stamina

	cured := v.cureHero()

	if changed || cured > 0 {
		v.Infof("NPC HEAL npc=%q class=%d life=%d->%d mana=%d->%d stamina=%.0f->%.0f cured=%d", npc.Label(), class,
			before.Life, after.Life, before.Mana, after.Mana, before.Stamina, after.Stamina, cured)
	}
}

// cureHero removes the states a healer removes and returns how many it removed.
func (v *Game) cureHero() int {
	if v.skills == nil || v.localPlayer == nil {
		return 0
	}

	curable := func(name string) bool {
		if v.asset == nil || v.asset.Records == nil {
			return false
		}

		r := v.asset.Records.States[name]

		return r != nil && r.Cureable
	}

	return v.skills.CureUnit(v.localPlayer.ID(), curable)
}

// talkSubmenu is the Talk row's submenu for an NPC with nothing to say from the quest system
// (UI_TalkTopicMenu 0x4b20f0): "introduction" unless the NPC has none, then "gossip". It reports whether it
// opened. The Jerhyn "about the merchants" row and the Cain cube row are not offered (UNVERIFIED actions).
func (v *Game) talkSubmenu(npc d2interface.MapEntity) bool {
	class := v.npcClassID(npc)
	rows := make([]d2player.NPCMenuRow, 0, 3)

	// scripted runs (OD2_AUTOSCRIPT) repeat Talk on one NPC and wait for its menu to come back: no submenu there
	if v.autoScript != nil || os.Getenv("OD2_AUTOSCRIPT") != "" {
		return false
	}

	for _, r := range d2npc.TalkRows(class, false) {
		switch r {
		case d2npc.TalkIntroduction:
			rows = append(rows, d2player.NPCMenuRow{StringID: 0xd47, Key: "Intro", Fallback: "introduction", Action: d2player.NPCActionIntro})
		case d2npc.TalkGossip:
			rows = append(rows, d2player.NPCMenuRow{StringID: 0xd43, Key: "NPCGossipMenu", Fallback: "gossip", Action: d2player.NPCActionGossip})
		}
	}

	if len(rows) == 0 {
		return false
	}

	menu := v.gameControls.NPCMenu
	menu.Open(npc.Label(), rows, 0, 0, func(row d2player.NPCMenuRow) { v.onNPCMenuChoice(npc, row) })
	v.anchorNPCMenu(menu, npc)

	v.Infof("NPC talk submenu opened: npc=%q class=%d rows=%d", npc.Label(), class, len(rows))

	return true
}

// sayNPCLine voices and shows one line of the NPC's pool.
func (v *Game) sayNPCLine(npc d2interface.MapEntity, class, index int, what string) {
	e, ok := d2npc.EntryFor(class)
	if !ok || index < 0 || index >= len(e.Records) {
		v.Infof("NPC %s: %q has no line %d", what, npc.Label(), index)

		return
	}

	msg := e.Records[index].Msg
	text := ""

	if key := d2npc.MessageKey(msg); key != "" {
		if t := v.asset.TranslateString(key); t != "" && t != key {
			text = speechText(t)
		}
	}

	snd, hasSound := d2quest.SoundForMessage(msg)

	v.Infof("NPC SPEECH %s npc=%q class=%d index=%d msg=%d sound=%d key=%s text=%q", what, npc.Label(), class, index, msg,
		snd.Index, d2npc.MessageKey(msg), shorten(text, 60))
	v.logSubtitle(npc.Label(), msg, text)

	if hasSound && speechPlays() {
		v.playSpeech(snd.Index)
	}

	if text != "" {
		v.gameControls.Speech.Show(npc.Label(), text, speechSeconds(text))
	}

	v.npcTown().gate.Spoke(class, v.rerollGossip)
}

// npcIntro plays the "introduction" line (callback 0x4b0a00).
func (v *Game) npcIntro(npc d2interface.MapEntity) {
	class := v.npcClassID(npc)

	e, ok := d2npc.EntryFor(class)
	if !ok {
		return
	}

	v.gameControls.NPCMenu.Close()
	v.sayNPCLine(npc, class, d2npc.IntroIndex(e, v.heroExeClass()), "introduction")
}

// npcGossip plays the gossip line chosen for the NPC (callback 0x4b09e0).
func (v *Game) npcGossip(npc d2interface.MapEntity) {
	class := v.npcClassID(npc)

	// the first speech of the game rolls every NPC; make sure a line is chosen before reading it
	t := v.npcTown()
	if t.gate.Chosen(class) == 0 {
		t.gate.Spoke(class, v.rerollGossip)
	}

	v.gameControls.NPCMenu.Close()
	v.sayNPCLine(npc, class, t.gate.Chosen(class), "gossip")
}
