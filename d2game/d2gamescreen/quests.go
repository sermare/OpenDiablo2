package d2gamescreen

import (
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2act3"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2drlg"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2hireling"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2reward"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapgen"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
	"github.com/OpenDiablo2/OpenDiablo2/d2game/d2player"
)

const (
	questFrameRate     = 25.0 // the original's frame rate; quest timers count frames
	speechSecondsBase  = 2.0
	speechCharsPerSec  = 14.0
	questNoticeSeconds = 4.0
)

// questItemCodes are the item codes the quest system tracks in the inventory.
//
//nolint:gochecknoglobals // static lookup data
var questItemCodes = []string{
	d2quest.ItemHoradricMalus, d2quest.ItemScrollOfInifuss, d2quest.ItemDecipheredScroll,
	d2quest.ItemHoradricScroll, d2quest.ItemBookOfSkill,
	d2quest.ItemHoradricCube, d2quest.ItemStaffOfKingsShaft, d2quest.ItemViperAmulet, d2quest.ItemHoradricStaff,
	d2quest.ItemLamEsenTome, d2quest.ItemKhalimEye, d2quest.ItemKhalimHeart, d2quest.ItemKhalimBrain,
	d2quest.ItemKhalimFlail, d2quest.ItemKhalimWill, d2quest.ItemGidbinn, d2quest.ItemJadeFigurine,
	d2quest.ItemGoldenBird, d2quest.ItemMephistoSoulstone, d2quest.ItemHellforgeHammer, d2quest.ItemMalahScroll, d2quest.ItemDefrostPotion,
}

// questRuntime joins the pure quest system (d2quest) to the running game.
type questRuntime struct {
	g        *d2quest.Game
	frameAcc float64
	area     int
	dirty    bool // the quest log needs a refresh
	logSeen  map[int]int
	auto     *autoQuest
	// lastNPC is the NPC whose dialog is open (for closing it when the player leaves).
	lastNPC d2interface.MapEntity
	// imbuePending and respec record effects the engine has no UI for yet.
	// spoken lists the message ids voiced so far; topics the last topic list
	// (both for the autotest).
	spoken       []int
	topics       []d2quest.Speech
	denFoes      map[*d2mapentity.Monster]bool // monsters a scripted run spawned for the Den of Evil
	barkAcc      float64
	barkWait     float64 // seconds until Navi may bark again
	imbuePending bool
	respec       bool
	rogueHire    bool
	// rewards is the state of the quest rewards (Larzuk, Anya, Malah...).
	rewards d2reward.State
	// actPortalLevel is the level in which the act portal was opened (0: none).
	actPortalLevel int
}

// d2sClass converts the engine's hero enum to the .d2s class number.
func d2sClass(h d2enum.Hero) int {
	switch h {
	case d2enum.HeroAmazon:
		return d2quest.ClassAmazon
	case d2enum.HeroSorceress:
		return d2quest.ClassSorceress
	case d2enum.HeroNecromancer:
		return d2quest.ClassNecromancer
	case d2enum.HeroPaladin:
		return d2quest.ClassPaladin
	case d2enum.HeroBarbarian:
		return d2quest.ClassBarbarian
	case d2enum.HeroDruid:
		return d2quest.ClassDruid
	case d2enum.HeroAssassin:
		return d2quest.ClassAssassin
	}

	return d2quest.ClassSorceress
}

// questAreaAlias is OD2_AUTOQUEST_AREA: the levels.txt id the quest system
// treats the current map as (OD2_REALMAPS level 9 is a Cave Level 1 that
// stands in for the Den of Evil, level 8, which has no DRLG preset yet).
func questAreaAlias() int {
	n, _ := strconv.Atoi(os.Getenv("OD2_AUTOQUEST_AREA"))

	return n
}

// quests returns the quest runtime, creating it once the hero exists.
func (v *Game) quests() *questRuntime {
	if v.questRT != nil {
		return v.questRT
	}

	if v.localPlayer == nil || v.gameControls == nil {
		return nil
	}

	p := v.localPlayer
	if p.Progress == nil {
		p.Progress = &d2hero.HeroProgress{}
	}

	diff := p.QuestDifficulty
	if n, err := strconv.Atoi(os.Getenv("OD2_AUTOQUEST_DIFF")); err == nil && n >= 0 && n <= 2 {
		diff = n
	}

	r := &questRuntime{logSeen: map[int]int{}, area: d2quest.LevelRogueEncampment}
	v.questRT = r

	if os.Getenv("OD2_AUTOQUEST") != "" {
		r.auto = newAutoQuest(os.Getenv("OD2_AUTOQUEST"))
		r.auto.resetRecord(p.Progress, diff)
	}

	r.g = d2quest.New(p.Progress.QuestRecord(diff), &p.Progress.NPC, diff)
	r.g.Hero = d2quest.Hero{Class: d2sClass(p.Class), Level: p.Stats.Level}
	r.g.Trace = func(s string) { v.Infof("%s", s) }

	for code, n := range v.gameControls.ItemCountsByCode() {
		for _, q := range questItemCodes {
			if code == q {
				r.g.Items[code] = n
			}
		}
	}

	r.g.Start()
	v.Infof("QUEST system started difficulty=%d class=%d level=%d (Acts 1-5)",
		diff, r.g.Hero.Class, r.g.Hero.Level)
	v.Infof("HERO state at start: level=%d exp=%d skillpoints=%d statpoints=%d gold=%d", p.Stats.Level, p.Stats.Experience,
		p.Stats.SkillPoints, p.Stats.StatsPoints, p.Gold)

	for _, l := range r.g.Log() {
		v.Infof("QUEST LOG act=%d quest=%d status=%d page=%d view=%d/%d/%s", l.Act, l.Index, l.Status, l.Page,
			l.View.Cat, l.View.Page, l.View.Key)
	}

	r.dirty = true

	v.gameControls.QuestLog().SetOnCompletionSeen(func(act, index int) {
		r.g.LogSeen(act, index) // client packet 0x58: sets the UPDATEQUESTLOG bit
		r.dirty = true
	})

	// the welcome-back flags of the save arm the NPC return greetings
	for _, class := range r.g.ReturnGreetingNPCs() {
		if name := d2player.NPCClassName(class); name != "" {
			if v.returnGreet == nil {
				v.returnGreet = returnGreetings{}
			}

			v.returnGreet.Arm(strings.ToLower(name))
		}
	}

	// the area the map stands for (a scripted run starts in town and moves
	// between areas itself)
	area := d2mapgen.RealLevel()
	if a := questAreaAlias(); a != 0 {
		area = a
	}

	if r.auto != nil {
		area = 0
	}

	if area != 0 && area != r.area {
		v.questArea(area)
	}

	if v.monsters != nil {
		v.monsters.OnKill = v.onMonsterKilled
	}

	return r
}

// restartQuests builds the quest engine again on the (just cleared) record. The quests that were done when
// the game started are dormant in the old engine, so a debug reset of the record alone would leave Mephisto,
// the Council and the rest deaf to their events.
func (v *Game) restartQuests() {
	r := v.questRT
	if r == nil || v.localPlayer == nil {
		return
	}

	p, old := v.localPlayer, r.g
	g := d2quest.New(old.Rec, old.NPC, old.Difficulty)
	g.Hero, g.Trace, g.Level = old.Hero, old.Trace, old.Level

	r.g = g
	v.syncQuestItems()
	g.Start()

	r.dirty = true

	v.Infof("QUEST system restarted on the cleared record (difficulty=%d, level=%d)", g.Difficulty, p.Stats.Level)
}

// advanceQuests runs the quest timers, the log refresh and the OD2_AUTOQUEST scenario.
func (v *Game) advanceQuests(elapsed float64) {
	r := v.quests()
	if r == nil {
		return
	}

	if v.monsters != nil && v.monsters.OnKill == nil {
		v.monsters.OnKill = v.onMonsterKilled
	}

	r.frameAcc += elapsed * questFrameRate
	if r.frameAcc >= 1 {
		n := int(r.frameAcc)
		r.frameAcc -= float64(n)
		v.applyQuestEffects(r.g.Tick(n))
	}

	if r.dirty {
		v.syncQuestLog()
	}

	v.advanceBarks(elapsed)
	v.advanceUber(elapsed)
	v.advanceRite(elapsed)
	v.advanceHellforge(elapsed)
	v.advanceChaos(elapsed)
	v.advanceAct3(elapsed)

	if r.auto != nil {
		r.auto.advance(engineHost{v}, elapsed)
	}
}

// questDispatch feeds an event to the quest system and applies the effects.
func (v *Game) questDispatch(e d2quest.Event) {
	r := v.quests()
	if r == nil {
		return
	}

	r.g.Hero.Level = v.localPlayer.Stats.Level
	v.applyQuestEffects(r.g.Dispatch(e))

	r.dirty = true
	v.maybeOpenActPortal()
}

// questArea tells the quest system the hero moved to another area.
func (v *Game) questArea(to int) {
	r := v.quests()
	if r == nil || to == r.area {
		return
	}

	from := r.area
	r.area = to

	v.Infof("QUEST area %d -> %d", from, to)
	v.questDispatch(d2quest.Event{Kind: d2quest.EvAreaChanged, OldLevel: from, NewLevel: to})
}

// onMonsterKilled is the monster director's death hook.
func (v *Game) onMonsterKilled(ev d2monsters.KillEvent) {
	r := v.quests()
	if r == nil {
		return
	}

	if ev.Class == d2quest.NPCBaalCrab && r.area == d2quest.LevelWorldstoneChamber && v.heroExpansion() {
		v.playBaalMovie()
	}

	// the Den of Evil is cleared when no monster is left in the level
	if r.area == d2quest.LevelDenOfEvil && v.monsters != nil {
		alive := 0

		for _, m := range v.monsters.Monsters() {
			// a scripted run (OD2_AUTOQUEST_REAL) only counts the monsters it spawned
			if m.Alive() && (len(r.denFoes) == 0 || r.denFoes[m]) {
				alive++
			}
		}

		r.g.SyncDenMonsters(alive)

		// the cleared Den's sting comes with the hero line 0x23 the quest effects send
	}

	if id, ok := questKillVoices[ev.Label]; ok && ev.ByHero {
		v.Infof("QUEST kill voice %q id=%#x", ev.Label, id)
		v.sayHero(id)
	} else if h, ok := questStingers[ev.Label]; ok && ev.ByHero {
		v.playStinger(h, "killed "+ev.Label)
	}

	super := ""
	if strings.HasPrefix(ev.Label, "The ") || strings.Contains(ev.Label, "Countess") || d2quest.IsQuestSuper(ev.Label) {
		super = ev.Label
	}

	// the Council of Travincal: only the three members count for the Blackened Temple, not their followers
	class, isSuper := ev.Class, false
	if ev.Monster != nil && ev.Monster.SuperUnique != "" {
		isSuper = true
		if super == "" {
			super = ev.Monster.SuperUnique
		}
	}

	if ev.Monster != nil {
		v.act3.killX, v.act3.killY = ev.Monster.SubtilePos()
	}

	freed := 0

	switch {
	case class == d2quest.NPCPrisonDoor && v.monsters != nil && ev.Monster != nil:
		// -1: a door that freed nobody (0 would stand for a full group of five in the scripted walk)
		if freed = v.monsters.RescueDoorDied(ev.Monster); freed == 0 {
			freed = -1
		}
	case class == d2quest.NPCGidbinnBoss && ev.Monster != v.act3.bossMon:
		class = 0 // an ordinary fetish of the same class
	case class == prisonerMonClass && v.monsters != nil && ev.Monster != nil:
		// only a freed barbarian that dies counts (the exe lists the freed ones)
		if !v.monsters.RescuePrisonerDied(ev.Monster) {
			class = 0
		}
	}

	v.questDispatch(d2quest.Event{Kind: d2quest.EvMonsterKilled, Monster: d2act3.QuestKillClass(class, isSuper), Super: super, Name: ev.Label, Level: r.area, Freed: freed})
	v.questKillDrops(ev.Label, ev.Class)

	if ev.ByHero && d2quest.KillForcesSave(ev.Class) {
		v.Infof("SAVE forced by the kill of %s (act boss save point)", ev.Label)
		v.saveBeforeExit()
	}
	v.act3Killed(ev)
	v.riteKilled(ev)
	v.uberKilled(ev)
	v.trioKilled(ev)
	v.chaosKilled(ev)
}

// questObjectOperated is a quest object (cairn stone, Malus chest...) used by the hero.
func (v *Game) questObjectOperated(ob *d2mapentity.Object) {
	if id := ob.Record().Index; isSealObject(id) {
		v.chaosOperate(ob)

		return
	}

	id := ob.Record().Index

	if v.anyaOperate(ob) {
		return
	}

	// the chests, the altar and the tome give their item once: they open (animation) and a second click
	// does nothing, like the original's object state
	switch id {
	// (Lam Esen's tome and the Gidbinn objects are opened by act3Operate before it calls this: a second Open here
	// would report "already open" and the quest system would never hear of the operation)
	case d2quest.ObjectCubeChest, d2quest.ObjectScrollChest, d2quest.ObjectStaffChest, d2quest.ObjectTaintedSunAltar:
		opened, err := ob.Open()
		if err != nil {
			v.Warningf("opening quest object %q: %v", ob.Label(), err)
		}

		if !opened {
			return
		}
	}

	v.Infof("QUEST object operated id=%d name=%q", id, ob.Label())
	v.syncQuestItems()

	if id == d2quest.ObjectOrifice && d2mapgen.RealMapsEnabled() {
		// the maze generator puts the orifice only into the tomb the game seed draws (d2drlg.RealTomb)
		v.questRT.g.RealTomb = d2drlg.RealTomb(d2mapgen.HeroMapSeed)
	}

	v.questDispatch(d2quest.Event{Kind: d2quest.EvObjectOperated, Object: id, Level: v.quests().area, Wielded: v.wieldedCode()})
}

// syncQuestItems raises the quest system's count of every quest item to what the inventory holds: items that
// arrived by a console command, a trade or a cube recipe do not raise a pick-up event, and an object that wants an
// item (the Compelling Orb wants Khalim's Will) must see it.
func (v *Game) syncQuestItems() {
	r := v.questRT
	if r == nil || v.gameControls == nil {
		return
	}

	for code, n := range v.gameControls.ItemCountsByCode() {
		for _, q := range questItemCodes {
			if code == q && r.g.Items[code] < n {
				r.g.Items[code] = n
			}
		}
	}
}

// questItemPickedUp reports an item the hero picked up.
func (v *Game) questItemPickedUp(code string) {
	code = strings.TrimSpace(code)

	for _, q := range questItemCodes {
		if code == q {
			v.questDispatch(d2quest.Event{Kind: d2quest.EvItemPickedUp, Item: code})
			return
		}
	}
}

// applyQuestEffects does what the quest system asked for.
func (v *Game) applyQuestEffects(effects []d2quest.Effect) {
	r := v.questRT

	for _, e := range effects {
		switch e.Kind {
		case d2quest.EffectLogUpdate:
			v.Infof("QUEST EFFECT log-update quest=%d page=%d", e.Quest, e.Value)

			r.dirty = true
		case d2quest.EffectSkillPoint:
			v.localPlayer.Stats.SkillPoints += e.Value
			v.Infof("QUEST EFFECT skill-point +%d total=%d", e.Value, v.localPlayer.Stats.SkillPoints)
		case d2quest.EffectHireRogues:
			r.rogueHire = true
			v.Infof("QUEST EFFECT hire-rogues (%s)", e.Note)
			v.grantQuestMerc(d2hireling.SellerKashya)
		case d2quest.EffectImbue:
			r.imbuePending = true
			v.Infof("QUEST EFFECT imbue-available (%s)", e.Note)
		case d2quest.EffectGiveItem:
			v.Infof("QUEST EFFECT give-item code=%s quality=%d ilvl/count=%d (%s)", e.Code, e.Quality, e.Value, e.Note)
			v.spawnQuestItem(e.Code, questItemOptions(v.asset.Records.Item.All[e.Code], e.Quality, e.Value)...)
		case d2quest.EffectDeleteItem:
			ok := v.gameControls.RemoveItemByCode(e.Code)
			v.Infof("QUEST EFFECT delete-item code=%s removed=%v", e.Code, ok)

			if isHellforgeSmash(e) {
				v.startHellforgeShower()
			}
		case d2quest.EffectSpawn:
			switch {
			case e.Code == d2quest.GidbinnBossCode:
				v.Infof("QUEST EFFECT spawn-boss code=%s (%s)", e.Code, e.Note)
				v.spawnGidbinnBoss()
			case e.Code == d2quest.GidbinnDropCode:
				v.Infof("QUEST EFFECT spawn-drop code=%s (%s)", d2quest.ItemGidbinn, e.Note)
				v.dropQuestItemAt(d2quest.ItemGidbinn, v.act3.killX, v.act3.killY)
			case e.Code != "":
				v.Infof("QUEST EFFECT spawn-item code=%s (%s)", e.Code, e.Note)
				v.spawnQuestItem(e.Code)
			default:
				v.Infof("QUEST EFFECT spawn (%s) [not simulated]", e.Note)
			}
		case d2quest.EffectRespec:
			r.respec = true
			v.Infof("QUEST EFFECT respec-available (Akara, slot 41)")
		case d2quest.EffectSound:
			// the value is the hero voice id of UNITSND_PlayUnitSoundById (0x13 impossible .. 0x53 defeat_baal)
			v.Infof("QUEST EFFECT sound id=%d quest=%d (%s)", e.Value, e.Quest, e.Note)
			v.sayHero(e.Value)
		case d2quest.EffectPortal:
			v.questPortal(e)
		case d2quest.EffectUnlockAct:
			v.Infof("QUEST EFFECT unlock act %d", e.Value)
		case d2quest.EffectBark:
			v.Infof("QUEST EFFECT bark msg=%d", e.Value)
		case d2quest.EffectReward:
			v.applyQuestReward(e)
		}
	}
}

// questItemEffects is what reading or drinking a quest item does: the quest game checks the reward bit and
// returns the effects, or none when the item does nothing now (the item then stays in the inventory). The bool
// says whether the code is one of the three usable quest items.
func questItemEffects(g *d2quest.Game, code string) ([]d2quest.Effect, bool) {
	switch strings.TrimSpace(code) {
	case d2quest.ItemBookOfSkill:
		return g.ReadBookOfSkill(), true
	case d2quest.ItemPotionOfLife:
		return g.DrinkPotionOfLife(), true
	case d2quest.ItemMalahScroll:
		return g.ReadScrollOfResistance(), true
	}

	return nil, false
}

// useQuestItem is the item-use hook of the inventory: it reports whether the item was a quest item that took
// effect (and so is consumed). An item whose reward bit is not set stays where it is.
func (v *Game) useQuestItem(code string) bool {
	r := v.quests()
	if r == nil {
		return false
	}

	effects, known := questItemEffects(r.g, code)
	if !known {
		return false
	}

	if len(effects) == 0 {
		v.Infof("QUEST item use code=%s: nothing happens (reward bit not set)", strings.TrimSpace(code))

		return false
	}

	v.Infof("QUEST item use code=%s", strings.TrimSpace(code))
	v.applyQuestEffects(effects)

	return true
}

// spawnQuestItem drops a quest reward at the hero's feet (the engine's reward
// items are ground items the hero picks up; an approximation of the original,
// which puts them in the inventory). Unknown item codes are logged, not sent.
func (v *Game) spawnQuestItem(code string, options ...string) {
	if v.asset.Records.Item.All[code] == nil {
		v.Infof("QUEST EFFECT item code %q is not in the item tables; nothing dropped", code)
		return
	}

	v.debugSpawnItemAtPlayer(append([]string{code}, options...)...)
}

// questItemOptions are the NewItem options of a quest reward: the quality
// (0 normal, 1 magic, 2 rare) and the item level; for stackable items the value
// is a count (eight stamina potions).
func questItemOptions(rec *d2records.ItemCommonRecord, quality, value int) []string {
	var out []string

	switch quality {
	case 1:
		out = append(out, "q=magic")
	case 2:
		out = append(out, "q=rare")
	}

	switch {
	case value <= 0:
	case rec != nil && rec.Stackable:
		out = append(out, "qty="+strconv.Itoa(value))
	default:
		out = append(out, "ilvl="+strconv.Itoa(value))
	}

	return out
}

// syncQuestLog pushes the quest states to the quest log panel.
func (v *Game) syncQuestLog() {
	r := v.questRT
	r.dirty = false

	statuses := map[int]int{}
	texts := map[int]d2player.QuestText{}
	changed := false

	for _, l := range r.g.Log() {
		key := d2player.QuestLogKey(l.Act, l.Index)

		// the icon class and the text come from the client rules of the original (d2quest.ClientLogView)
		st := d2enum.QuestStatusNotStarted

		switch l.View.Cat {
		case d2quest.LogCatAnimating:
			st = d2enum.QuestStatusCompleting // the log plays the completion animation once
		case d2quest.LogCatDone:
			st = d2enum.QuestStatusCompleted
		case d2quest.LogCatActive:
			st = l.View.Page
			if st < 1 {
				st = d2enum.QuestStatusInProgress
			}
		}

		statuses[key] = st
		texts[key] = d2player.QuestText{Key: l.View.Key, Count: l.View.Count, Append: l.View.Append}

		if old, seen := r.logSeen[key]; seen && old != st {
			changed = true

			v.Infof("QUEST LOG act=%d quest=%d status=%d page=%d view=%d/%d/%s (was %d)", l.Act, l.Index, l.Status, l.Page,
				l.View.Cat, l.View.Page, l.View.Key, old)
		}

		r.logSeen[key] = st
	}

	v.gameControls.QuestLog().SetTexts(texts)
	v.gameControls.QuestLog().SetStatuses(statuses)

	if changed {
		v.gameControls.Speech.Notice("Quest log updated", questNoticeSeconds)
	}
}

// questLogText returns what the quest log shows for a quest (autotest).
func (v *Game) questLogText(act, index int) string {
	return v.gameControls.QuestLog().DescriptionText(act, index)
}
