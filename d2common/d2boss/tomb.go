package d2boss

// Tomb is the Duriel encounter (act 2, quest A2Q6 "The Seven Tombs").
//
// VERIFIED (QUEST_A2_TheSevenTombs_OnMonsterKilled 0x59ac40, quest id 13, slot
// 14, the lair is level 0x49 = 73 in the area handler): when Duriel dies the
// quest state changes, a quest timer of 8 is started (Tyrael's arrival), the
// speech/sound 0x3d is played, the record bit 5 of slot 14 is set when none of
// the bits 0, 3, 4, 5 is, and the object stored by the quest (the orifice) gets
// mode 1.
//
// From the object table: the orifice is object 152 (OperateFn 25) and the
// "Portal to Duriel's Lair" is object 100 (OperateFn 43).
//
// UNVERIFIED: the delay between the staff and the portal, where Tyrael
// stands, and that the exit is a permanent town portal.
type Tomb struct {
	state tombState
	// DurielAlive is set between the spawn and the kill.
	DurielAlive bool
}

type tombState int

const (
	tombSealed tombState = iota
	tombStaffPlaced
	tombPortalOpen
	tombInLair
	tombDone
)

var tombNames = [...]string{"sealed", "staff-placed", "portal-open", "in-lair", "done"}

// NewTomb creates the encounter.
func NewTomb() *Tomb { return &Tomb{} }

// Name implements Encounter.
func (t *Tomb) Name() string { return "duriel" }

// State implements Encounter.
func (t *Tomb) State() string { return tombNames[t.state] }

// OnOperate: the orifice takes the staff; the Duriel portal enters the lair.
func (t *Tomb) OnOperate(m *Manager, e Operate) {
	switch e.Object {
	case ObjOrifice:
		if t.state != tombSealed {
			return
		}

		if !e.HasStaff {
			m.logf("duriel trigger: orifice operated without the Horadric Staff: nothing happens")

			return
		}

		t.state = tombStaffPlaced
		m.logf("duriel trigger: Horadric Staff placed in the orifice -> state %s", t.State())
		m.emit("duriel", Action{Kind: ActObjectMode, Name: "orifice", Class: ObjOrifice, Note: "staff inserted, the tomb shakes"})
		m.after(tombPortalDelay, func() {
			t.state = tombPortalOpen
			m.logf("duriel state -> %s", t.State())
			m.emit("duriel", Action{Kind: ActPortal, Name: "portal to Duriel's Lair", Class: ObjDurielPortal,
				Level: LevelDurielLair, Note: "delay unverified"})
		})
	case ObjDurielPortal:
		if t.state == tombPortalOpen {
			m.logf("duriel trigger: the hero steps into the portal")
		}
	}
}

// OnEnter: entering the lair meets Duriel.
func (t *Tomb) OnEnter(m *Manager, level int) {
	if level != LevelDurielLair {
		return
	}

	if t.state == tombDone || t.DurielAlive {
		m.logf("duriel trigger: lair entered again (state %s): no new Duriel", t.State())

		return
	}

	t.state = tombInLair
	t.DurielAlive = true
	m.logf("duriel trigger: lair entered -> state %s", t.State())
	m.emit("duriel", Action{Kind: ActSpawnMonster, Name: "Duriel", Class: ClassDuriel, Super: -1, Boss: true,
		Note: "preset boss of the lair"})
}

// OnKill: Duriel's death.
func (t *Tomb) OnKill(m *Manager, k Kill) {
	if k.Class != ClassDuriel {
		return
	}

	t.DurielAlive = false
	t.state = tombDone
	m.logf("duriel killed -> state %s", t.State())
	m.emit("duriel", Action{Kind: ActMessage, Name: "Duriel dead", Msg: 0x3d, Note: "quest sound/speech 0x3d"})
	m.emit("duriel", Action{Kind: ActObjectMode, Name: "orifice", Class: ObjOrifice, Note: "mode 1 after the kill"})
	m.after(tyraelDelay, func() {
		m.emit("duriel", Action{Kind: ActSpawnNPC, Name: "Tyrael", Class: ClassTyrael, Super: -1, Note: "after the quest timer"})
		m.emit("duriel", Action{Kind: ActPortal, Name: "portal to Lut Gholein", Class: ObjTownPortal,
			Level: 40, Note: "object unverified"})
	})
}

// OnTick implements Encounter.
func (t *Tomb) OnTick(*Manager, int) {}

// LairWarpBlocked is the gate of the warp into Duriel's Lair (level 73):
// SERVER_EnterWarpTile 0x553140 asks FUN_00543a70, which for level 73 calls
// FUN_0059b700 = "the Seven Tombs node (id 13) is active and its private byte
// +0xb is 0" and cancels the warp then. VERIFIED: that test. INFERRED: that
// byte +0xb becomes non-zero when the staff is in the orifice (no writer was
// found in the quest's handlers 0x59a730, 0x59a390, 0x59adb0 or the helpers
// next to 0x59b700), so the lair is blocked until the staff is placed.
func (t *Tomb) LairWarpBlocked() bool { return t.state == tombSealed }
