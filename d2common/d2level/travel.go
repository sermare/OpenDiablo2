package d2level

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Cooldowns between level changes (session-core.md, verified): the take-
// waypoint handler drops requests made less than 10 s after the last level
// change, the portal operate function uses 5 s.
const (
	WaypointCooldownSeconds = 10.0
	PortalCooldownSeconds   = 5.0
)

// Portal jump aftermath (verified): after a portal jump the unit gets stat
// list/state 0x66 for 0x4B frames (25 frames per second).
const (
	PortalStateID     = 0x66
	PortalStateFrames = 75
	// FramesPerSecond is the server tick rate.
	FramesPerSecond = 25
)

// Cooldown remembers when the player last changed level. Times are seconds
// on any monotonic clock the caller chooses.
type Cooldown struct {
	last float64
	set  bool
}

// Ready reports whether at least window seconds passed since the last level
// change (always true before the first one).
func (c *Cooldown) Ready(now, window float64) bool {
	return !c.set || now-c.last >= window
}

// Mark records a level change at time now.
func (c *Cooldown) Mark(now float64) {
	c.last, c.set = now, true
}

// Portal errors.
var (
	ErrPortalCooldown  = errors.New("portal: less than 5 s since the last level change")
	ErrPortalNotOwner  = errors.New("portal: owned by a player outside your party")
	ErrPortalQuestFlag = errors.New("portal: the destination level is locked by a quest flag")
	ErrPortalNoDest    = errors.New("portal: no destination level")
)

// PortalRequest is a use of a town portal (objects 0x3b) or permanent portal
// (0x3c) object as SERVER_UsePortalObject (0x582700) sees it.
type PortalRequest struct {
	Dest int // destination level, from the portal's object data
	// Restricted is true when the game limits portals to their owner (and
	// party); then Owner/Player/InParty matter.
	Restricted bool
	Owner      string
	Player     string
	InParty    bool
	// QuestFlag is the Levels.txt QuestFlag (classic) / QuestFlagEx (expansion)
	// of the destination, 0 if none. HasQuestFlag reports whether the player's
	// quest record has it.
	QuestFlag    int
	HasQuestFlag func(flag int) bool
}

// ValidatePortal applies the portal rules; the cooldown is checked by the
// caller with Cooldown.Ready(now, PortalCooldownSeconds).
func ValidatePortal(r PortalRequest) error {
	if r.Dest <= 0 {
		return ErrPortalNoDest
	}

	if r.Restricted && r.Owner != r.Player && !r.InParty {
		return ErrPortalNotOwner
	}

	if r.QuestFlag != 0 && (r.HasQuestFlag == nil || !r.HasQuestFlag(r.QuestFlag)) {
		return ErrPortalQuestFlag
	}

	return nil
}

// WarpRange is the largest distance (tiles) between the player and a room tile
// for SERVER_InteractWithUnitByType case 5 (stairs, doors, warps): the
// original requires distance < 5 (0x642980). (Waypoint objects use their own
// operate range.)
const WarpRange = 5.0

// LoadAct is the S2C 0x03 packet that tells the client to rebuild an act.
type LoadAct struct {
	Act        int    // 1..5 (the packet carries 0..4)
	Seed       uint32 // game seed (pGame+0x7c)
	StartLevel uint16 // start level of the act
	Aux        uint32 // pGame+0x80, the object-region seed
}

// LoadActPacketSize is the size of the packet in bytes.
const LoadActPacketSize = 12

const loadActID = 0x03

// Encode writes [0x03][act u8][seed u32][startLevel u16][aux u32], little
// endian (layout verified, session-core.md).
func (p LoadAct) Encode() []byte {
	b := make([]byte, LoadActPacketSize)
	b[0] = loadActID
	b[1] = byte(p.Act - 1)
	binary.LittleEndian.PutUint32(b[2:], p.Seed)
	binary.LittleEndian.PutUint16(b[6:], p.StartLevel)
	binary.LittleEndian.PutUint32(b[8:], p.Aux)

	return b
}

// DecodeLoadAct parses a LoadAct packet.
func DecodeLoadAct(b []byte) (LoadAct, error) {
	if len(b) != LoadActPacketSize || b[0] != loadActID {
		return LoadAct{}, fmt.Errorf("not a LoadAct packet (%d bytes)", len(b))
	}

	if b[1] > NumActs-1 {
		return LoadAct{}, fmt.Errorf("LoadAct: act %d out of range", b[1])
	}

	return LoadAct{
		Act:        int(b[1]) + 1,
		Seed:       binary.LittleEndian.Uint32(b[2:]),
		StartLevel: binary.LittleEndian.Uint16(b[6:]),
		Aux:        binary.LittleEndian.Uint32(b[8:]),
	}, nil
}

// Plan is what SERVER_ChangePlayerLevel decides for one level change.
type Plan struct {
	From, To  int
	FromAct   int
	ToAct     int
	ActChange bool
	StartType StartType
	// TownTransition is set when SERVER_HandleTownTransition has work to do:
	// the old or the new level is a town.
	TownTransition bool
	// LoadAct is the packet the client needs for an act change (nil within an
	// act, where the unit is just moved to a room of the new level).
	LoadAct *LoadAct
}

// ErrBadLevel is returned for level ids outside 1..132 (Levels.txt rows that
// exist in the game).
var ErrBadLevel = errors.New("invalid level id")

// PlanTransition mirrors SERVER_ChangePlayerLevel: within an act the unit is
// moved with the given start type; across acts the destination act is built
// on demand from the same seed and the client gets a LoadAct for it. The aux
// seed is the second per-game seed (pGame+0x80).
func PlanTransition(from, to int, start StartType, seed, aux uint32) (Plan, error) {
	if to < 1 || to > 132 {
		return Plan{}, fmt.Errorf("%w: %d", ErrBadLevel, to)
	}

	p := Plan{
		From: from, To: to,
		FromAct: ActOfLevel(from), ToAct: ActOfLevel(to),
		StartType:      start,
		TownTransition: IsTown(from) || IsTown(to),
	}

	if p.FromAct != p.ToAct {
		p.ActChange = true
		p.LoadAct = &LoadAct{Act: p.ToAct, Seed: seed, StartLevel: uint16(ActStartLevel(p.ToAct)), Aux: aux}
	}

	return p, nil
}

// ActWorlds tracks which act dungeons exist. The original creates the
// dungeon of an act the first time a player enters it (SERVER_EnsureActDungeon)
// from the same game seed, so (seed, act, difficulty) fully determines it.
type ActWorlds struct {
	built [NumActs]bool
}

// Ensure marks the act as built and reports whether this call built it.
func (a *ActWorlds) Ensure(act int) bool {
	if act < 1 || act > NumActs || a.built[act-1] {
		return false
	}

	a.built[act-1] = true

	return true
}

// Built reports whether the act has been built.
func (a *ActWorlds) Built(act int) bool {
	return act >= 1 && act <= NumActs && a.built[act-1]
}
