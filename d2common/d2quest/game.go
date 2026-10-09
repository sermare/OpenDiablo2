// Package d2quest implements the quest system of the original game as a pure,
// engine independent state machine: a list of quest nodes, a set of events
// the engine feeds in (area entered, monster killed, item picked up, NPC
// talked to, message heard...), the 16 flag bits of the d2s quest record
// they drive, and the effects (skill point, mercenary, imbue, items...) the
// engine has to apply.
//
// The model follows the 1.14b binary notes (d2-re-notes/quests.md and
// quests-2.md) and the clean-room D2MOO sources the notes compare against.
// Differences to the real game, all deliberate:
//
//   - one player per game: the party loops (members in the same room, party
//     members elsewhere in the act) collapse to "the hero"; the per-player
//     GUID lists of the original become booleans on the quest;
//   - object, missile and portal effects are reported as Effects instead of
//     being simulated;
//   - quests that are not implemented (Act 2 after Radament and Acts 3-5) have
//     no node; their record slots are left untouched.
//
// Evidence: the state machines of Act 1 and Radament's Lair follow the clean-room
// D2MOO quest sources, which the notes checked against the 1.14b binary for the
// speech tables (all tables, byte for byte) and for the A1Q1/A1Q2/A2Q1 message
// handlers. UNVERIFIED against the binary: the other quests' handlers, the 1.14b
// only deltas (the Akara respec bits are from the binary notes, the Uber quests
// are not modelled), the ids of the attach-sound effects, the topic captions of
// the Talk submenu, the heard-list handling of the client, the barking distance,
// and the Charsi message 150 mode (the binary dump says topic, D2MOO says spoken;
// the binary table is used).
//
// Bit meanings are in d2s.QuestBit*; see the constants below for the names
// the quest code uses.
package d2quest

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Flag bits (D2C_OriginalQuestFlags names, quests-2.md section 0).
const (
	FlagRewardGranted  = d2s.QuestBitDone          // 0
	FlagRewardPending  = d2s.QuestBitRewardPending // 1
	FlagStarted        = 2
	FlagLeaveTown      = 3
	FlagEnterArea      = 4
	FlagCustom1        = 5
	FlagCustom2        = 6
	FlagCustom3        = 7
	FlagCustom4        = 8
	FlagCustom6        = 10
	FlagUpdateLog      = d2s.QuestBitClosed // 12
	FlagPrimaryGoal    = d2s.QuestBitUpdated
	FlagCompletedNow   = d2s.QuestBitGoalReached
	FlagCompletedEarly = d2s.QuestBitCompleteOnEntry
)

// EventKind is the kind of an event; the numeric values are the event ids of
// the original quest nodes (quests.md section 2.2).
type EventKind int

// The events a quest node can handle.
const (
	EvNpcActivate   EventKind = 0  // the player clicked an NPC (build the speech list)
	EvNpcDeactivate EventKind = 2  // the player finished talking
	EvAreaChanged   EventKind = 3  // Old/NewLevel
	EvItemPickedUp  EventKind = 4  // Item
	EvItemDropped   EventKind = 5  // Item
	EvItemRemoved   EventKind = 6  // Item left the game (deleted, sold...)
	EvMonsterKilled EventKind = 8  // Monster / Super / Level
	EvLeftGame      EventKind = 10 // player left
	EvMessageHeard  EventKind = 11 // NPC + Msg: the client acknowledged an NPC message
	EvGameStarted   EventKind = 13 // restore state from the flag bits
	// EvObjectOperated is not an event id of the binary (the original has a
	// separate object dispatch, QUEST_OnObjectOperated 0x5428c0): Object is the
	// objects.txt row.
	EvObjectOperated EventKind = 15
	numEvents                  = 16
)

// Event is one thing that happened in the game.
type Event struct {
	Kind               EventKind
	NPC                int    // NPC monster class (activate, deactivate, heard)
	Msg                int    // message id (heard)
	OldLevel, NewLevel int    // area changes
	Monster            int    // monster class (killed)
	Super              string // super unique name ("The Countess") when the monster is one
	Name               string // display name of the monster that died ("Duriel")
	Level              int    // level where the monster died / the object was operated
	Item               string // item code without padding ("hdm")
	Object             int    // objects.txt row
}

// EffectKind says what the engine has to do.
type EffectKind int

// The effects quests ask the engine for.
const (
	// EffectLogUpdate: the quest log page of Quest changed to Value.
	EffectLogUpdate EffectKind = iota
	// EffectSkillPoint: add Value skill points (Den of Evil).
	EffectSkillPoint
	// EffectHireRogues: Kashya's rogue mercenaries become hirable.
	EffectHireRogues
	// EffectImbue: Charsi may imbue one item (Tools of the Trade reward).
	EffectImbue
	// EffectGiveItem: create Code with Quality (0 normal, 1 magic, 2 rare) and
	// item level Value on the hero.
	EffectGiveItem
	// EffectDeleteItem: remove one Code from the hero's inventory.
	EffectDeleteItem
	// EffectSound: play the quest sound Value (35 = Den of Evil done, 36 =
	// Horadric Malus found, 37 = Countess; ids from the original attach-sound
	// calls, meaning of the ids unverified).
	EffectSound
	// EffectRespec: Akara offers the free stat/skill reset (slot 41).
	EffectRespec
	// EffectPortal: a town portal / the portal to Tristram opens (Note says which).
	EffectPortal
	// EffectBark: an overhead speech bubble, Value is the message id.
	EffectBark
	// EffectSpawn: spawn an NPC or object (Note describes it: Cain at the gibbet...).
	EffectSpawn
	// EffectUnlockAct: a new act can be travelled to (Value = act number 1..5).
	EffectUnlockAct
)

// Effect is a request to the engine.
type Effect struct {
	Kind    EffectKind
	Quest   int
	Value   int
	Code    string
	Quality int
	Note    string
}

// Change records one bit change for the tracing/autotest log.
type Change struct {
	Quest      int
	Slot       int
	Before     uint16
	After      uint16
	Reason     string
	StateAfter int
}

// Hero is what quests need to know about the player.
type Hero struct {
	Class int // .d2s class number (ClassAmazon...)
	Level int
}

type timer struct {
	due int
	fn  func()
}

// Quest is one quest node.
type Quest struct {
	ID    int
	Slot  int // record slot
	Act   int // 0..4
	Name  string
	Label string // "A1Q1" prefix in the message table
	// LogIndex is the position (1..6) in the quest log of its act; 0 for quests
	// that have no log entry (prologues, intro and Navi).
	LogIndex int

	State     int  // fState: shared game state of the quest
	LastState int  // fLastState: the quest log page last pushed
	Active    bool // bActive
	NotIntro  bool // bNotIntro: false for the intro quests and for inert (completed) quests
	InitNo    int  // states below this show the stored log page (nInitNo)
	SeqID     int  // quest made available when this one completes (-1: none)
	// NoSetState quests (prologue, Navi, intros) are never made inert at game start.
	NoSetState bool

	tables   [][]Speech
	on       [numEvents]func(g *Game, q *Quest, e *Event)
	activate func(g *Game, q *Quest, npc int) []Speech
	active   func(g *Game, q *Quest, npc int) bool
	status   func(g *Game, q *Quest) int
	seq      func(g *Game, q *Quest) bool
	data     interface{}
}

// Table returns speech table i (nil when out of range).
func (q *Quest) Table(i int) []Speech {
	if i < 0 || i >= len(q.tables) {
		return nil
	}

	return q.tables[i]
}

// Game holds the quest state of one running game and hero.
type Game struct {
	Rec        *d2s.QuestRecord // the hero's quest record (normal/nightmare/hell)
	NPC        *d2s.NPCBlock    // intro flags; may be nil
	Difficulty int
	Hero       Hero
	// Expansion selects the Lord of Destruction rules (the Cow King checks the
	// Baal quest instead of the Diablo quest).
	Expansion bool
	// Level is the id of the level the hero is in.
	Level int
	// Items counts the quest items the hero carries by item code.
	Items map[string]int
	// Trace, if set, receives every state change as a line.
	Trace func(string)
	// Rand drives the cairn stone order; fixed by tests.
	Rand *rand.Rand

	Quests  []*Quest
	byID    map[int]*Quest
	effects []Effect
	changes []Change
	global  d2s.QuestRecord // QUESTS_SetGlobalState: shared flags of the game
	frame   int
	timers  []timer
	heard   []int // the client's list of heard messages (9 entries in the original)

	denSpawned, denKilled int
	denKnown              bool

	dialog *Dialog
}

const heardCap = 9

// New creates the quest system for a hero. rec must not be nil. The caller
// sets Hero, Level and Items, then calls Start.
func New(rec *d2s.QuestRecord, npc *d2s.NPCBlock, difficulty int) *Game {
	g := &Game{
		Rec: rec, NPC: npc, Difficulty: difficulty,
		Hero: Hero{Class: ClassSorceress, Level: 1}, Level: LevelRogueEncampment,
		Items: map[string]int{}, byID: map[int]*Quest{}, Expansion: true,
		Rand: rand.New(rand.NewSource(1)), //nolint:gosec // not crypto
	}

	for _, init := range []func() *Quest{
		newA1Prologue, newDenOfEvil, newBurialGrounds, newToolsOfTheTrade, newSearchForCain,
		newForgottenTower, newSistersToTheSlaughter, newA2Prologue, newRadament, newHoradricStaff,
		newTaintedSun, newArcaneSanctuary, newSummoner, newNavi,
		newA1Intro,
	} {
		q := init()
		g.Quests = append(g.Quests, q)
		g.byID[q.ID] = q
	}

	for _, q := range newBossQuests() {
		g.Quests = append(g.Quests, q)
		g.byID[q.ID] = q
	}

	return g
}

// Quest returns a quest node by id.
func (g *Game) Quest(id int) *Quest { return g.byID[id] }

// Start is the player-joined-game sequence (QUEST_SyncPlayerOnGameEnter and
// QUESTS_SequenceCycler): completed quests become inert, every node restores
// its state from the flag bits, and the quest chains are walked so that the
// quest after a completed one is available.
func (g *Game) Start() {
	// QUESTREC_LoadFromBuffer: the transient bits are dropped and a pending
	// reward becomes "completed before"
	for slot := 0; slot < d2s.QuestSlots; slot++ {
		g.Rec.Clear(slot, FlagPrimaryGoal)
		g.Rec.Clear(slot, FlagCompletedNow)

		if g.Rec.Get(slot, FlagRewardPending) {
			g.Rec.Set(slot, FlagCompletedEarly)
		}
	}

	for _, q := range g.Quests {
		if q.NoSetState || q.Slot > d2s.QuestSlots {
			continue
		}

		if g.Rec.Get(q.Slot, FlagRewardGranted) || g.Rec.Get(q.Slot, FlagCompletedEarly) {
			q.NotIntro, q.Active, q.LastState = false, false, 0
			g.global.Set(q.Slot, FlagCompletedEarly)
		}
	}

	for _, q := range g.reversed() {
		if f := q.on[EvGameStarted]; f != nil {
			f(g, q, &Event{Kind: EvGameStarted})
		}
	}

	for _, id := range []int{QuestDenOfEvil, QuestRadament} {
		if q := g.byID[id]; q != nil && q.seq != nil {
			q.seq(g, q)
		}
	}
}

// reversed returns the nodes newest first, the order the original walks them
// (pLastQuest -> pPrev).
func (g *Game) reversed() []*Quest {
	out := make([]*Quest, len(g.Quests))
	for i, q := range g.Quests {
		out[len(g.Quests)-1-i] = q
	}

	return out
}

// Dispatch feeds an event to all quest nodes and returns the effects raised.
func (g *Game) Dispatch(e Event) []Effect {
	switch e.Kind {
	case EvAreaChanged:
		g.Level = e.NewLevel
	case EvItemPickedUp:
		g.Items[e.Item]++
	case EvItemRemoved, EvItemDropped:
		if g.Items[e.Item] > 0 {
			g.Items[e.Item]--
		}
	}

	actFilter := e.Kind == EvNpcActivate || e.Kind == EvNpcDeactivate || e.Kind == EvMessageHeard

	for _, q := range g.reversed() {
		f := q.on[e.Kind]
		if f == nil || (actFilter && q.Act != actOfLevel(g.Level)) {
			continue
		}

		ev := e
		f(g, q, &ev)
	}

	return g.TakeEffects()
}

// TakeEffects returns and clears the pending effects.
func (g *Game) TakeEffects() []Effect {
	out := g.effects
	g.effects = nil

	return out
}

// TakeChanges returns and clears the recorded bit changes.
func (g *Game) TakeChanges() []Change {
	out := g.changes
	g.changes = nil

	return out
}

// Tick advances the quest timers by frames (25 per second).
func (g *Game) Tick(frames int) []Effect {
	g.frame += frames

	for {
		fired := false

		for i, t := range g.timers {
			if t.due <= g.frame {
				g.timers = append(g.timers[:i], g.timers[i+1:]...)
				t.fn()

				fired = true

				break
			}
		}

		if !fired {
			break
		}
	}

	return g.TakeEffects()
}

func (g *Game) after(frames int, fn func()) {
	g.timers = append(g.timers, timer{due: g.frame + frames, fn: fn})
}

func (g *Game) emit(e Effect) { g.effects = append(g.effects, e) }

func (g *Game) tracef(format string, args ...interface{}) {
	if g.Trace != nil {
		g.Trace(fmt.Sprintf(format, args...))
	}
}

// ---- flag helpers ----

func (g *Game) get(q *Quest, bit int) bool { return g.Rec.Get(q.Slot, bit) }

func (g *Game) record(q *Quest, before uint16, reason string) {
	after := g.Rec.Slot(q.Slot)
	if after == before {
		return
	}

	g.changes = append(g.changes, Change{Quest: q.ID, Slot: q.Slot, Before: before, After: after, Reason: reason, StateAfter: q.State})
	g.tracef("QUEST %s slot=%d bits 0x%04x->0x%04x (%s) state=%d", q.Label, q.Slot, before, after, reason, q.State)
}

func (g *Game) set(q *Quest, bit int, reason string) {
	before := g.Rec.Slot(q.Slot)
	g.Rec.Set(q.Slot, bit)
	g.record(q, before, reason)
}

func (g *Game) clear(q *Quest, bit int, reason string) {
	before := g.Rec.Slot(q.Slot)
	g.Rec.Clear(q.Slot, bit)
	g.record(q, before, reason)
}

func (g *Game) resetProgress(q *Quest, reason string) {
	before := g.Rec.Slot(q.Slot)
	g.Rec.ClearProgressBits(q.Slot)
	g.record(q, before, reason)
}

// setState is QUESTS_StateDebug: it changes the quest state.
func (g *Game) setState(q *Quest, n int) {
	if q.State != n {
		g.tracef("QUEST %s state %d->%d", q.Label, q.State, n)
	}

	q.State = n
}

func (g *Game) hasItem(code string) bool { return g.Items[code] > 0 }

// globalDone is QUESTS_SetGlobalState(PRIMARYGOALDONE).
func (g *Game) globalDone(q *Quest) { g.global.Set(q.Slot, FlagPrimaryGoal) }

// GlobalDone reports the shared primary-goal flag of a quest.
func (g *Game) GlobalDone(q *Quest) bool { return g.global.Get(q.Slot, FlagPrimaryGoal) }

// cycle is QUESTS_UnitIterate(q, page, ..., iterate) followed by the status
// cycler: the quest log page is stored in LastState and, when it should be
// shown, pushed to the client.
func (g *Game) cycle(q *Quest, page int, push bool) {
	q.LastState = page

	if !push {
		return
	}

	// ACT1Qn_UnitIterate_StatusCyclerEx: push if the quest is not yet granted
	// (and not completed earlier), or the goal / completion flag is set.
	if (!g.get(q, FlagRewardGranted) && !g.get(q, FlagCompletedEarly)) ||
		g.get(q, FlagPrimaryGoal) || g.get(q, FlagCompletedNow) {
		g.emit(Effect{Kind: EffectLogUpdate, Quest: q.ID, Value: g.LogPage(q)})
	}
}

// LogPage is the quest log page byte the server sends for a quest
// (QUESTS_RefreshStatus, or the quest's own status filter).
func (g *Game) LogPage(q *Quest) int {
	if q.status != nil {
		return q.status(g, q)
	}

	return g.refreshStatus(q)
}

func (g *Game) refreshStatus(q *Quest) int {
	completedNow := g.get(q, FlagCompletedNow)

	if q.State < q.InitNo {
		if !completedNow {
			if q.ID != QuestCain || q.LastState != 6 {
				return q.LastState
			}

			if !g.get(q, FlagPrimaryGoal) {
				return 12
			}

			return q.LastState
		}

		return 12
	}

	if !g.get(q, FlagPrimaryGoal) {
		if q.ID == QuestCain && completedNow {
			if q.State == 6 {
				return 12
			}

			return q.LastState
		}

		return 12
	}

	return q.LastState
}

// LogStatus is the state of a quest for the log panel.
type LogStatus int

// The states the quest log distinguishes.
const (
	LogNotStarted LogStatus = iota
	LogInProgress
	LogCompleting // goal reached, reward not claimed
	LogCompleted
)

// QuestLog is one quest as the log panel shows it.
type QuestLog struct {
	Quest  int
	Act    int // 1..5
	Index  int // 1..6 within the act
	Status LogStatus
	// Page is the description page (string qstsa<act>q<index><page>), 0 for none.
	Page int
	// Unseen is set for a completed quest whose completion the player has not
	// yet looked at in the log (bit 12 UPDATEQUESTLOG is clear).
	Unseen bool
}

// LogSeen records that the player saw the completion of the quest at (act
// 1..5, index 1..6) in the quest log; the client reports this with packet 0x58
// and the server sets the UPDATEQUESTLOG bit (D2MOO PlrMsg.cpp, Rcv0x58).
func (g *Game) LogSeen(act, index int) {
	for _, q := range g.Quests {
		if q.Act+1 == act && q.LogIndex == index && g.get(q, FlagRewardGranted) && !g.get(q, FlagUpdateLog) {
			g.set(q, FlagUpdateLog, "completion seen in the quest log")
		}
	}
}

// Log returns the log entries of the implemented real quests (Act 1 quests 1-6
// and Radament), computed from the current state.
func (g *Game) Log() []QuestLog {
	var out []QuestLog

	for _, q := range g.Quests {
		if q.LogIndex < 1 {
			continue
		}

		l := QuestLog{Quest: q.ID, Act: q.Act + 1, Index: q.LogIndex}

		switch {
		case g.get(q, FlagRewardGranted):
			l.Status = LogCompleted
			l.Unseen = !g.get(q, FlagUpdateLog)
		case g.get(q, FlagRewardPending) && g.get(q, FlagPrimaryGoal):
			l.Status, l.Page = LogCompleting, g.LogPage(q)
		default:
			l.Page = g.LogPage(q)
			if l.Page != 0 {
				l.Status = LogInProgress
			}
		}

		out = append(out, l)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Act != out[j].Act {
			return out[i].Act < out[j].Act
		}

		return out[i].Index < out[j].Index
	})

	return out
}

// Describe returns a one line state dump of a quest for logs.
func (g *Game) Describe(q *Quest) string {
	bits := make([]string, 0, 16)
	names := []string{"RG", "RP", "STARTED", "LEAVETOWN", "ENTERAREA", "C1", "C2", "C3", "C4", "C5", "C6", "C7", "LOG", "PGD", "CN", "CB"}
	v := g.Rec.Slot(q.Slot)

	for i, n := range names {
		if v>>uint(i)&1 != 0 {
			bits = append(bits, n)
		}
	}

	return fmt.Sprintf("%s slot=%d bits=0x%04x [%s] state=%d last=%d page=%d", q.Label, q.Slot, v, strings.Join(bits, "|"),
		q.State, q.LastState, g.LogPage(q))
}
