package d2gamescreen

import (
	"os"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2enum"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2interface"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2quest"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2reward"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2hero"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapentity"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2map/d2mapgen"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2monsters"
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
	d2quest.ItemGoldenBird, d2quest.ItemMephistoSoulstone, d2quest.ItemHellforgeHammer, d2quest.ItemMalahScroll,
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
		v.Infof("QUEST LOG act=%d quest=%d status=%d page=%d", l.Act, l.Index, l.Status, l.Page)
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
	}

	super := ""
	if strings.HasPrefix(ev.Label, "The ") || strings.Contains(ev.Label, "Countess") || d2quest.IsQuestSuper(ev.Label) {
		super = ev.Label
	}

	v.questDispatch(d2quest.Event{Kind: d2quest.EvMonsterKilled, Monster: ev.Class, Super: super, Name: ev.Label, Level: r.area})
	v.questKillDrops(ev.Label, ev.Class)
	v.uberKilled(ev)
}

// questObjectOperated is a quest object (cairn stone, Malus chest...) used by the hero.
func (v *Game) questObjectOperated(ob *d2mapentity.Object) {
	id := ob.Record().Index

	v.Infof("QUEST object operated id=%d name=%q", id, ob.Label())
	v.questDispatch(d2quest.Event{Kind: d2quest.EvObjectOperated, Object: id, Level: v.quests().area})
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
		case d2quest.EffectImbue:
			r.imbuePending = true
			v.Infof("QUEST EFFECT imbue-available (%s)", e.Note)
		case d2quest.EffectGiveItem:
			v.Infof("QUEST EFFECT give-item code=%s quality=%d ilvl/count=%d (%s)", e.Code, e.Quality, e.Value, e.Note)
			v.spawnQuestItem(e.Code)
		case d2quest.EffectDeleteItem:
			ok := v.gameControls.RemoveItemByCode(e.Code)
			v.Infof("QUEST EFFECT delete-item code=%s removed=%v", e.Code, ok)
		case d2quest.EffectSpawn:
			if e.Code != "" {
				v.Infof("QUEST EFFECT spawn-item code=%s (%s)", e.Code, e.Note)
				v.spawnQuestItem(e.Code)
			} else {
				v.Infof("QUEST EFFECT spawn (%s) [not simulated]", e.Note)
			}
		case d2quest.EffectRespec:
			r.respec = true
			v.Infof("QUEST EFFECT respec-available (Akara, slot 41)")
		case d2quest.EffectSound:
			v.Infof("QUEST EFFECT sound id=%d quest=%d (%s) [not played: the attach-sound table is not decoded]", e.Value, e.Quest, e.Note)
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

// spawnQuestItem drops a quest reward at the hero's feet (the engine's reward
// items are ground items the hero picks up; an approximation of the original,
// which puts them in the inventory). Unknown item codes are logged, not sent.
func (v *Game) spawnQuestItem(code string) {
	if v.asset.Records.Item.All[code] == nil {
		v.Infof("QUEST EFFECT item code %q is not in the item tables; nothing dropped", code)
		return
	}

	v.debugSpawnItemAtPlayer(code)
}

// syncQuestLog pushes the quest states to the quest log panel.
func (v *Game) syncQuestLog() {
	r := v.questRT
	r.dirty = false

	statuses := map[int]int{}
	changed := false

	for _, l := range r.g.Log() {
		key := d2player.QuestLogKey(l.Act, l.Index)

		st := d2enum.QuestStatusNotStarted

		switch l.Status {
		case d2quest.LogCompleted:
			st = d2enum.QuestStatusCompleted
			if l.Unseen {
				st = d2enum.QuestStatusCompleting // the log plays the completion animation once
			}
		case d2quest.LogInProgress, d2quest.LogCompleting:
			st = l.Page
			if st < 1 {
				st = d2enum.QuestStatusInProgress
			}
		}

		statuses[key] = st

		if old, seen := r.logSeen[key]; seen && old != st {
			changed = true

			v.Infof("QUEST LOG act=%d quest=%d status=%d page=%d (was %d)", l.Act, l.Index, l.Status, l.Page, old)
		}

		r.logSeen[key] = st
	}

	v.gameControls.QuestLog().SetStatuses(statuses)

	if changed {
		v.gameControls.Speech.Notice("Quest log updated", questNoticeSeconds)
	}
}

// questLogText returns what the quest log shows for a quest (autotest).
func (v *Game) questLogText(act, index int) string {
	return v.gameControls.QuestLog().DescriptionText(act, index)
}
