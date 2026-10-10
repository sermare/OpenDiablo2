package d2gamescreen

import (
	"fmt"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2reward"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/diablo2item"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

// The quest rewards through the NPCs themselves. The speech of the reward lines
// (Talk) comes from the quest system; the item rewards (Larzuk's sockets,
// Anya's personalisation, Charsi's imbue) are done when the hero clicks the NPC
// with an item on the cursor (OnItemDropOnNPC, or the walk of a script that
// holds one), and Akara's reset is a row of her menu (string 0x2ba0 of the
// original's table, gated by quest slot 41).

// rewardDrop is an item reward the hero walks to claim with the cursor item.
type rewardDrop struct {
	npc  d2interface.MapEntity
	kind d2reward.ItemKind
}

// itemRewardOwed reports whether the NPC's reward on an item is waiting.
func (v *Game) itemRewardOwed(k d2reward.ItemKind) bool {
	r := v.quests()
	if r == nil {
		return false
	}

	return d2reward.OwedFrom(&r.rewards, v.imbueOwed()).Of(k)
}

// withRewardRows adds the reward rows of the NPC to its menu: Akara's reset
// while the Den of Evil reset is available, and "Add Sockets" / "Personalize" /
// "Imbue" for Larzuk, Anya and Charsi while their reward is owed.
func (v *Game) withRewardRows(class int, rows []d2player.NPCMenuRow) []d2player.NPCMenuRow {
	out := append([]d2player.NPCMenuRow{}, rows...)

	if class == d2quest.NPCAkara && v.respecAvailable() {
		out = append(out, d2player.RowRespec)
	}

	if k, ok := d2reward.ItemKindOf(class); ok && v.itemRewardOwed(k) {
		out = append(out, d2player.NPCMenuRow{Fallback: k.Verb(), Action: d2player.NPCActionReward})
	}

	return out
}

// onRewardRow is the choice of a reward row.
func (v *Game) onRewardRow(npc d2interface.MapEntity, row d2player.NPCMenuRow) {
	v.gameControls.NPCMenu.Close()

	switch row.Action {
	case d2player.NPCActionRespec:
		if err := v.akaraRespec(); err != nil {
			v.Infof("REWARD respec refused: %v", err)
			v.gameControls.Speech.Notice(err.Error(), questNoticeSeconds)
		}
	case d2player.NPCActionReward:
		k, ok := d2reward.ItemKindOf(v.npcClassID(npc))
		if !ok {
			return
		}

		v.gameControls.OpenInventoryPanel()
		v.Infof("REWARD select kind=%s npc=%q: take an item from the inventory and click the NPC with it", k, npc.Label())
		v.gameControls.Speech.Notice(fmt.Sprintf("%s: take an item and click %s with it", k.Verb(), npc.Label()), questNoticeSeconds)
	}

	v.endConversationUnlessMenuOpen()
}

// OnItemDropOnNPC implements d2player.ItemDropListener: the hero clicked an NPC
// with an item on the cursor. It returns true when the NPC takes the click (the
// item then stays on the cursor and is not dropped to the ground).
func (v *Game) OnItemDropOnNPC(npc d2interface.MapEntity, _ d2player.InventoryItem) bool {
	return v.tryRewardDrop(npc)
}

// tryRewardDrop starts the claim of an item reward when the NPC performs one,
// it is owed and the cursor holds an item. The hero first walks up to the NPC.
func (v *Game) tryRewardDrop(npc d2interface.MapEntity) bool {
	if v.gameControls == nil || v.gameControls.CursorItem() == nil {
		return false
	}

	k, ok := d2reward.ItemKindOf(v.npcClassID(npc))
	if !ok || !v.itemRewardOwed(k) {
		return false
	}

	targetX, targetY := npc.GetPositionF()

	v.Infof("REWARD drop kind=%s npc=%q: walking up with the item on the cursor", k, npc.Label())
	v.gameControls.NPCMenu.Close()
	v.rewardDrop = &rewardDrop{npc: npc, kind: k}
	v.npcTarget = npc
	v.OnPlayerMove(targetX, targetY)

	return true
}

// finishRewardDrop is the hero standing at the NPC with the item.
func (v *Game) finishRewardDrop() {
	drop := v.rewardDrop
	v.rewardDrop = nil
	v.npcTarget = nil

	it, _ := v.gameControls.CursorItem().(*diablo2item.Item)
	if it == nil {
		v.Infof("REWARD drop kind=%s npc=%q: the cursor is empty", drop.kind, drop.npc.Label())

		return
	}

	msg, err := v.applyItemRewardTo(string(drop.kind), it)
	if err != nil {
		v.Infof("REWARD refused by %q: %v", drop.npc.Label(), err)
		v.gameControls.Speech.Notice(err.Error(), questNoticeSeconds)

		return
	}

	v.Infof("REWARD %s done by %q: %s", drop.kind, drop.npc.Label(), msg)
	v.gameControls.Speech.Notice(msg, questNoticeSeconds)
	v.playNPCGreeting(drop.npc.Label())
}

// ---- Akara: reset stat and skill points ----

// respecAvailable is the gate of Akara's row: the Den of Evil reset bits of
// quest slot 41 (REWARDPENDING, see d2s.QuestRecord.AkaraRespecAvailable).
func (v *Game) respecAvailable() bool {
	r := v.questRT
	if r == nil || r.g == nil || r.g.Rec == nil {
		return false
	}

	return r.g.Rec.AkaraRespecAvailable()
}

// akaraRespec resets the attributes to the class base and refunds every spent
// skill point (one use: the bit is cleared and the reward bit set, UNVERIFIED
// like the whole 1.13 free respec).
func (v *Game) akaraRespec() error {
	if !v.respecAvailable() {
		return fmt.Errorf("Akara has no reset to offer")
	}

	p := v.localPlayer
	st := p.Stats

	cs := v.asset.Records.Character.Stats[p.Class]
	if cs == nil || st == nil {
		return fmt.Errorf("no class record for the hero")
	}

	// only the skills of the class tree hold spent points; Attack and the other base skills do not
	token := strings.ToLower(p.Class.GetToken3())
	spent := 0

	for _, sk := range p.Skills {
		if sk != nil && sk.SkillRecord != nil && strings.EqualFold(sk.Charclass, token) {
			spent += sk.SkillPoints
		}
	}

	stat, skill := d2reward.Respec(
		d2reward.Allocation{Strength: st.Strength, Dexterity: st.Dexterity, Vitality: st.Vitality, Energy: st.Energy, SkillPoints: spent},
		d2reward.Allocation{Strength: cs.InitStr, Dexterity: cs.InitDex, Vitality: cs.InitVit, Energy: cs.InitEne})

	st.Strength, st.Dexterity, st.Vitality, st.Energy = cs.InitStr, cs.InitDex, cs.InitVit, cs.InitEne
	st.StatsPoints += stat
	st.SkillPoints += skill

	for _, sk := range p.Skills {
		if sk != nil && sk.SkillRecord != nil && strings.EqualFold(sk.Charclass, token) {
			sk.SetPoints(0)
		}
	}

	rec := v.questRT.g.Rec
	rec.Clear(d2s.QuestSlotAkaraRespec, d2quest.FlagRewardPending)
	rec.Set(d2s.QuestSlotAkaraRespec, d2quest.FlagRewardGranted)

	v.recalcHero()
	st.Health, st.Mana = st.MaxHealth, st.MaxMana
	v.questRT.dirty = true
	v.questRT.respec = false

	v.Infof("REWARD respec by Akara: %d stat points and %d skill points returned (unused now stats=%d skills=%d)",
		stat, skill, st.StatsPoints, st.SkillPoints)
	v.gameControls.Speech.Notice(fmt.Sprintf("%d stat and %d skill points returned", stat, skill), questNoticeSeconds)
	v.gameControls.RefreshSkills()
	v.gameControls.SaveItems()

	return nil
}

// ---- console aids for the scenarios ----

// commandQuestPending is "questpending <act> <quest>": puts the quest into the
// state of a finished quest whose reward waits (REWARDPENDING and PRIMARYGOAL,
// no REWARDGRANTED), so that the quest giver's Talk row speaks the reward line
// and claims it, as after the kill. Debug aid; local copy of the record only.
func (v *Game) commandQuestPending(args []string) error {
	var act, quest int

	if _, err := fmt.Sscan(args[0]+" "+args[1], &act, &quest); err != nil {
		return fmt.Errorf("invalid quest %s %s", args[0], args[1])
	}

	slot, ok := d2s.QuestSlot(act, quest)
	if !ok {
		return fmt.Errorf("invalid quest %d %d", act, quest)
	}

	if v.quests() == nil || v.questRT.g == nil {
		return fmt.Errorf("the quest system is not running")
	}

	rec := v.questRT.g.Rec
	rec.Clear(slot, d2quest.FlagRewardGranted)
	rec.Set(slot, d2quest.FlagRewardPending)
	rec.Set(slot, d2quest.FlagPrimaryGoal)

	v.Infof("QUEST act=%d quest=%d slot=%d reward pending (bits 0x%04x)", act, quest, slot, rec.Slot(slot))

	return nil
}

// commandPickItem is "pickitem <code|any>": takes the first inventory item
// the reward of the NPC in question accepts (or with this base code) onto the
// cursor, as a click would.
func (v *Game) commandPickItem(args []string) error {
	code := strings.ToLower(args[0])

	it := v.gameControls.PickItem(func(i *diablo2item.Item) bool {
		return code == "any" || strings.EqualFold(strings.TrimSpace(i.CommonCode), code)
	})
	if it == nil {
		return fmt.Errorf("no item %q in the inventory (or the cursor is busy)", code)
	}

	v.Infof("CURSOR picked up %s", strings.TrimSpace(it.CommonCode))

	return nil
}

// commandPutItem is "putitem": the cursor item goes back into the inventory.
func (v *Game) commandPutItem(_ []string) error {
	if !v.gameControls.PutCursorItemAway() {
		return fmt.Errorf("nothing on the cursor, or no room")
	}

	v.Infof("CURSOR item put away")

	return nil
}

// commandGiveItemQ is "giveitemq <code> <quality>" (see GiveItemQuality).
func (v *Game) commandGiveItemQ(args []string) error {
	var q int

	if _, err := fmt.Sscan(args[1], &q); err != nil {
		return fmt.Errorf("invalid quality %q", args[1])
	}

	name, err := v.giveRewardTestItem(args[0], q)
	if err != nil {
		return err
	}

	v.Infof("GIVEITEM code=%s quality=%d name=%q", args[0], q, name)

	return nil
}

// commandFreeInv is "freeinv <n>": makes room in a full inventory (debug).
func (v *Game) commandFreeInv(args []string) error {
	var n int

	if _, err := fmt.Sscan(args[0], &n); err != nil {
		return fmt.Errorf("invalid count %q", args[0])
	}

	v.Infof("FREEINV removed %d items", v.gameControls.FreeInventory(n))

	return nil
}
