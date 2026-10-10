package d2monster

import (
	"math"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// Command types stored in the AiGeneral command queue.
const (
	// CmdAlert is the group alert command (type 1) broadcast by Fallen
	// leaders, Scarabs and Minions (VERIFIED as a value; payload semantics
	// UNVERIFIED, see Command).
	CmdAlert = 1
	// CmdAnchor is the "home position" command (type 10) used by NPCs and
	// Blood Raven (VERIFIED value).
	CmdAnchor = 10
)

// Command is one node of the AiGeneral command queue (0x1c bytes in the exe:
// +8 type, +0xc/+0x10 position/target, +0x14 count, +0x18 delay).
type Command struct {
	Type   int
	X, Y   int
	Target uint32
	// Count is the node's +0x14 field. For CmdAlert it is treated as the
	// number of attacks the command still allows (UNVERIFIED: Minion AI uses
	// the same field as an expiry frame).
	Count int
	Delay int
}

// Brain is the per-monster AI state: AiGeneral plus the unit fields the AI
// reads. The owner (engine or test) keeps X, Y, Mode and HPPercent current.
type Brain struct {
	ID    uint32
	Class int
	Diff  Difficulty

	Profile *Profile
	Seed    *d2rand.Seed

	// Unit state maintained by the owner.
	X, Y      int // position in subtiles
	Size      int // unit size used by the edge distance
	Mode      Mode
	HPPercent int // 0..100
	LevelID   int // levels.txt id of the area the monster is in
	// Aggressive is FUN_005dbff0 (VERIFIED 0x5dbff0/0x571320): the monster's
	// pUnitData+0x54 is 3 or 0x13 (1, i.e. false, for a non-monster or a unit
	// without data). Which game events set 3/0x13 is not recorded.
	Aggressive bool

	// WakeShouted is AiGeneral flag 0x10, the one-shot Summoner wake-up
	// (VERIFIED 0x5aed10).
	WakeShouted bool

	// CanTeleport is AiGeneral flag 0x20 (+8 ushort, test 0x5dbf60), the gate
	// of the wounded MonTeleport (0x5aedc0, VERIFIED). Which code sets it is
	// UNVERIFIED, so it stays false (feature off) unless the host sets it.
	CanTeleport bool

	// TargetID/HasTarget record the target of the last acquisition (AiGeneral
	// +0x08); Tick keeps them current.
	TargetID  uint32
	HasTarget bool

	// AiGeneral.
	Def     *AIDef
	Scratch [3]int // +0x14, +0x18, +0x1c (zeroed whenever the AI state is set)
	// SpawnClass is AiGeneral +0x3c, the class a GenericSpawner picked (-1 =
	// none yet; set by its pre-hook).
	SpawnClass int
	queue      []Command
	Leader     *Brain
	Minions    []*Brain

	// Wake is the first frame at which Tick runs the AI again. Any action or
	// sleep issued by a think function pushes it out; the owner calls WakeNow
	// when the requested mode ends.
	Wake int

	// Forced-state bookkeeping (forced.go): the active forced state, the frame
	// it ends, the unit that caused it, and the AI to restore.
	Forced       ForcedKind
	ForcedUntil  int
	ForcedSource uint32
	baseDef      *AIDef
	// formationInit is the State 3 Pre-hook having run (reset by SetAI).
	formationInit bool
	// preRan is a pre-hook (MONAI_Pre_*) having run since the AI was set
	// (ai_faithful_a.go runs them lazily at the first think).
	preRan bool

	// Allied is true for a converted monster: it fights for the hero
	// (alignment 1 + the owner's target bucket in the exe). OwnerID is the
	// owning player's target id.
	Allied  bool
	OwnerID uint32
	// Attracting marks a monster cursed with Attract: other monsters treat it
	// as a target (target bucket 9 in the exe).
	Attracting bool
	// Airborne is the Vulture's flight flag (collision layer 5 in the exe).
	Airborne bool

	// Aux is a second generator for the coin flips the exe takes from a
	// global source (not the unit's seed), so that those never disturb the
	// unit's own roll sequence.
	Aux *d2rand.Seed
}

// NewBrain creates a brain for a unit. The unit seed is derived from the game
// seed and the unit id; the real derivation (UNIT_InitCreatedUnit) is not
// recorded in the notes, so this one is UNVERIFIED but deterministic.
func NewBrain(id uint32, class int, diff Difficulty, p *Profile, gameSeed uint32) *Brain {
	b := &Brain{
		ID:        id,
		Class:     class,
		Diff:      diff,
		Profile:   p,
		Seed:      d2rand.New(gameSeed + id),
		Mode:      ModeNeutral,
		HPPercent: 100,
		Size:      1,
		Aux:       d2rand.New(gameSeed*0x9E3779B1 + id + 0x51ED),
	}

	if def, ok := Lookup(p.AI); ok {
		b.Def = def
	} else {
		b.Def = unimplementedDef(p.AI)
	}

	return b
}

// SetAI switches the AI definition and clears the scratch state and the
// command queue, as MONAI_SetAiState does (VERIFIED).
func (b *Brain) SetAI(def *AIDef) {
	b.Def = def
	b.Scratch = [3]int{}
	b.SpawnClass = 0
	b.queue = nil
	b.formationInit = false
	b.preRan = false
}

// Roll advances the unit's own RNG and returns a value in [0, n).
func (b *Brain) Roll(n int) int { return int(b.Seed.Roll(int32(n))) }

// Chance is the percent test used by every think function: Roll(100) < p.
// It always consumes one RNG step.
func (b *Brain) Chance(p int) bool { return b.Roll(100) < p }

// AIP returns aip<n> (1..8) for the unit's profile.
func (b *Brain) AIP(n int) int { return b.Profile.AIP[n] }

// WakeNow makes the AI run on the next Tick (the unit's action finished).
func (b *Brain) WakeNow(frame int) { b.Wake = frame }

// PushCommand adds a command at the head of the queue.
func (b *Brain) PushCommand(c Command) { b.queue = append([]Command{c}, b.queue...) }

// PeekCommand returns the head of the queue.
func (b *Brain) PeekCommand() *Command {
	if len(b.queue) == 0 {
		return nil
	}

	return &b.queue[0]
}

// PopCommand removes the head of the queue.
func (b *Brain) PopCommand() {
	if len(b.queue) > 0 {
		b.queue = b.queue[1:]
	}
}

// FindCommand returns the first queued command of a type, or nil
// (FUN_0058ccf0: find the queue node whose +8 field equals the argument).
func (b *Brain) FindCommand(typ int) *Command {
	for i := range b.queue {
		if b.queue[i].Type == typ {
			return &b.queue[i]
		}
	}

	return nil
}

// AppendCommand adds a command at the tail of the queue (used for the anchor,
// which must not shadow a group alert at the head).
func (b *Brain) AppendCommand(c Command) { b.queue = append(b.queue, c) }

// QueueLen is the number of queued commands.
func (b *Brain) QueueLen() int { return len(b.queue) }

// AddMinion links m to b as in MONAI_AddMinionToLeader.
func (b *Brain) AddMinion(m *Brain) {
	m.Leader = b
	b.Minions = append(b.Minions, m)
}

// RemoveMinion unlinks m (MONAI_RemoveMinionFromLeader).
func (b *Brain) RemoveMinion(m *Brain) {
	for i, x := range b.Minions {
		if x == m {
			b.Minions = append(b.Minions[:i], b.Minions[i+1:]...)
			break
		}
	}

	m.Leader = nil
}

// IsGroupLeader reports whether b leads a group. In the exe the test is
// MONAI_GetLeaderUnit()==self; how a lone leader satisfies it is UNVERIFIED,
// so a leader here is a unit with at least one minion.
func (b *Brain) IsGroupLeader() bool { return b.Leader == nil && len(b.Minions) > 0 }

// Broadcast pushes a copy of c onto every minion's queue (VERIFIED,
// MONAI_BroadcastCommandToGroup).
func (b *Brain) Broadcast(c Command) {
	for _, m := range b.Minions {
		m.PushCommand(c)
	}
}

// DistanceTo returns the edge distance from b to a point.
func (b *Brain) DistanceTo(x, y int) int { return EdgeDistance(b.X-x, b.Y-y, b.Size) }

// waitForever is the Wake value while an action is in progress.
const waitForever = math.MaxInt32
