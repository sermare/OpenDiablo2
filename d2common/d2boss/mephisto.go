package d2boss

// Mephisto is the act 3 encounter (quest A3Q6 "The Guardian", id 20, slot 22).
//
// VERIFIED (QUEST_A3_TheGuardian_OnMonsterKilled 0x5ba440): the kill plays the
// sound 0x3e, sets the per-player quest flag 0xd (the "primary goal" bit of
// slot 22), starts a quest timer of 0xc and puts the stored object (the bridge
// / portal) in mode 1. The bridge before the lair is the object 341 (quest
// object 0x155): operating it with the quest in state 2 freezes the hero for
// the cutscene (FUN_005ba630); the exit after the kill is the hellgate portal,
// object 342.
//
// UNVERIFIED: Mephisto sleeps until a hero is within WakeRange (the AI itself
// only has the usual aggro radius; a boss wake-up delay of 20 frames exists in
// MONAI_PostTargetChecks for the Summoner, whether Mephisto shares it is
// unknown), the lair position and the portal delay.
type Mephisto struct {
	state mephState
	// WakeRange is the distance at which Mephisto wakes.
	WakeRange int
	// X, Y is Mephisto's resting spot (subtiles).
	X, Y int
}

type mephState int

const (
	mephIdle mephState = iota
	mephAsleep
	mephAwake
	mephDead
)

var mephNames = [...]string{"idle", "asleep", "awake", "dead"}

// NewMephisto creates the encounter.
func NewMephisto() *Mephisto { return &Mephisto{WakeRange: 25} }

// Name implements Encounter.
func (e *Mephisto) Name() string { return "mephisto" }

// State implements Encounter.
func (e *Mephisto) State() string { return mephNames[e.state] }

// OnEnter: the third Durance level places the sleeping Mephisto.
func (e *Mephisto) OnEnter(m *Manager, level int) {
	if level != LevelDurance3 || e.state != mephIdle {
		return
	}

	e.state = mephAsleep
	m.logf("mephisto trigger: Durance of Hate level 3 entered -> state %s", e.State())
	m.emit("mephisto", Action{Kind: ActSpawnMonster, Name: "Mephisto", Class: ClassMephisto, Super: -1, Boss: true,
		Asleep: true, X: e.X, Y: e.Y, Note: "preset boss, asleep"})
}

// OnOperate: the bridge cutscene.
func (e *Mephisto) OnOperate(m *Manager, o Operate) {
	if o.Object == ObjMephistoBridge && e.state == mephAsleep {
		m.logf("mephisto trigger: bridge operated (quest open=%v): cutscene, then he wakes", o.QuestOpen)

		e.wake(m, "bridge")
	}
}

func (e *Mephisto) wake(m *Manager, why string) {
	e.state = mephAwake
	m.logf("mephisto state -> %s (%s)", e.State(), why)
	m.emit("mephisto", Action{Kind: ActWake, Name: "Mephisto", Class: ClassMephisto, Note: why})
}

// OnTick: the hero walks up to him.
func (e *Mephisto) OnTick(m *Manager, _ int) {
	if e.state == mephAsleep && m.heroDist(e.X, e.Y) <= e.WakeRange {
		e.wake(m, "hero within the wake range")
	}
}

// OnKill: the portal home and the quest.
func (e *Mephisto) OnKill(m *Manager, k Kill) {
	if k.Class != ClassMephisto {
		return
	}

	e.state = mephDead
	m.logf("mephisto killed -> state %s", e.State())
	m.emit("mephisto", Action{Kind: ActMessage, Name: "Mephisto dead", Msg: 0x3e, Note: "quest sound/speech 0x3e"})
	m.after(hellgateDelay, func() {
		m.emit("mephisto", Action{Kind: ActPortal, Name: "red portal to the Pandemonium Fortress", Class: ObjHellgate,
			Level: LevelFortress, Note: "delay unverified"})
	})
}
