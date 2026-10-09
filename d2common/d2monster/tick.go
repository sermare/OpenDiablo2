package d2monster

import "strings"

// Target modes of the AI table (field +0 of the 16-byte entries).
const (
	TargetNone     = 0 // do not acquire a target (NPCs, sentries)
	TargetStandard = 1 // acquire the nearest hostile or sleep (VERIFIED)
	TargetOnly     = 2 // acquire only: the think function runs without a target if none
	// TargetFindOrWait (exe mode 4, VERIFIED at 0x5af2a5/0x5dd7f0, SandMaggot):
	// acquire; with none the tick sleeps 20 frames and ends.
	TargetFindOrWait = 4
	// TargetFindThenThink (exe mode 5, VERIFIED at 0x5af2cb, FrogDemon): the
	// same acquisition, but the think function runs even with no target.
	TargetFindThenThink = 5
)

// waitNoTargetFrames is the sleep 0x5dd7f0 schedules when modes 4 and 5 find
// nobody (VERIFIED: frame + 20).
const waitNoTargetFrames = 20

// summonerClass is the monstats id whose first sight of a player triggers the
// wake-up shout (VERIFIED immediate 0xfa in 0x5aed10).
const summonerClass = 250

// wakeUpFrames is the sleep after the wake-up shout (VERIFIED, push 0x14).
const wakeUpFrames = 20

// StateStunLike is the unit state that makes the pre-tick checks sleep 3
// frames (VERIFIED value 0x15; which game state that is, is not recorded).
const StateStunLike = 0x15

// Think is a think function (MONAI_Think_*).
type Think func(c *Ctx)

// AIDef is one row of the AI table (VERIFIED shape: targetMode, think).
type AIDef struct {
	Name       string
	TargetMode int
	Think      Think
	// Implemented is false for AIs listed only so the engine can tell the
	// user; their think function just sleeps.
	Implemented bool
}

var registry = map[string]*AIDef{}

func register(name string, mode int, t Think) {
	registry[strings.ToLower(name)] = &AIDef{Name: name, TargetMode: mode, Think: t, Implemented: true}
}

// Lookup returns the AI table entry for a monai/monstats AI name (case
// insensitive).
func Lookup(name string) (*AIDef, bool) {
	d, ok := registry[strings.ToLower(name)]

	return d, ok
}

// Implemented lists the names of the AIs this package implements.
func Implemented() []string {
	names := make([]string, 0, len(registry))
	for _, d := range registry {
		names = append(names, d.Name)
	}

	return names
}

// unimplementedDef stands in for AIs that are not ported: the monster idles.
func unimplementedDef(name string) *AIDef {
	return &AIDef{Name: name, TargetMode: TargetNone, Think: func(c *Ctx) { c.Sleep(25) }}
}

// Tick runs one scheduled AI event for a monster if it is due. It is the
// MONAI_RunThinkTick envelope (VERIFIED order): pre-tick checks, target
// acquisition by target mode, post-target checks, think function.
// It reports whether the think function ran.
func Tick(w World, b *Brain) bool {
	b.expireForced(w.Frame())

	if w.Frame() < b.Wake {
		return false
	}

	if !b.Mode.IsAlive() {
		b.Wake = waitForever

		return false
	}

	c := &Ctx{B: b, W: w}

	// 1. MONAI_PreTickChecks: state 0x15 sleeps 3 frames (VERIFIED). Opening
	// doors (monstats opendoors + path flag 0x800) and pursuing a remembered
	// unit (AiGeneral+0x0c) are NOT ported: the latter's setter is unknown.
	if w.HasState(b, StateStunLike) {
		c.Sleep(3)

		return false
	}

	// 2. MONAI_AcquireTargetByMode.
	if !acquire(c) {
		return false
	}

	// 3. MONAI_PostTargetChecks 0x5aefc0 (VERIFIED order: wake-up 0x5aed10,
	// wounded MonTeleport 0x5aedc0, threat re-targeting). Only the wake-up is
	// ported; the other two are not (see the notes, verify-monster-ai.md).
	if postTargetWake(c) {
		return false
	}

	// 4. The think function.
	b.Def.Think(c)

	if !c.acted {
		// The original always ends a think by scheduling something; a think
		// that did nothing would spin, so idle for a short while.
		c.Sleep(10)
	}

	return true
}

// acquire fills c.Params. It returns false when the tick ends (no target).
func acquire(c *Ctx) bool {
	mode := c.B.Def.TargetMode

	switch mode {
	case TargetNone:
		return true
	case TargetStandard, TargetOnly, TargetFindOrWait, TargetFindThenThink:
	default:
		return true
	}

	t, nearest, ok := pickTarget(c)

	// 0x5dc560 accepts a player only when its distance is strictly below the
	// aggro radius (VERIFIED: the compare branches away on "not below").
	thr := c.B.Profile.Aggro()
	if ok && nearest < thr {
		c.Target = &t
		c.B.TargetID, c.B.HasTarget = t.ID, true
		c.Dist = nearest
		c.InRange = c.W.InRange(c.B, t, nearest)

		return true
	}

	c.B.HasTarget = false

	switch mode {
	case TargetOnly, TargetFindThenThink:
		if mode == TargetFindThenThink {
			c.Sleep(waitNoTargetFrames)
		}

		return true
	case TargetFindOrWait:
		c.Sleep(waitNoTargetFrames)

		return false
	}

	// MONAI_AcquireStandardTargetOrSleep (VERIFIED): nothing to hunt, sleep
	// 25 if the nearest player is farther than 34, else (dist-10) if more
	// than 24, else 10. Which distance is used when no player is within the
	// aggro radius is the nearest one regardless of radius (VERIFIED: the min
	// distance 0x5dc560 reports is updated before its radius filter).
	// The exe also wanders (Wander(5)) here when the unit is aggressive and its
	// class has monstats flag bit 2 of the byte at +0xf0 (0x467af0, name
	// unknown) - not ported. (flag meaning
	// unknown).
	switch {
	case !ok || nearest > 34:
		c.Sleep(25)
	case nearest > 24:
		c.Sleep(nearest - 10)
	default:
		c.Sleep(10)
	}

	return false
}

// postTargetWake is MONAI_PostTargetChecks' first stage 0x5aed10 (VERIFIED):
// for a Summoner (class 250), or a unit the host marks "unaware" (the exe
// tests a game flag, FUN_0059dd60 with 8, whose meaning is UNVERIFIED), the
// first time a player is the target while the AiGeneral counter (+0x14) is
// below 20 and the one-shot flag 0x10 is clear: shout, set the flag, sleep 20
// frames and end the tick. (The older note read "dist < 20"; the exe compares
// the counter, not the distance.)
func postTargetWake(c *Ctx) bool {
	b := c.B
	if c.Target == nil || !c.Target.IsPlayer || b.WakeShouted || b.Scratch[0] >= 20 {
		return false
	}

	if b.Class != summonerClass && !(b.Def != nil && b.Def.Name == "Summoner") {
		return false
	}

	if s, ok := c.W.(Shouter); ok {
		s.Shout(b)
	}

	b.WakeShouted = true
	c.Sleep(wakeUpFrames)

	return true
}

// findStandard is the target search of 0x5dd6b0 without its idle side effects
// (VERIFIED: it returns the nearest player strictly inside the aggro radius,
// else schedules a sleep or wander the caller then overrides).
func (c *Ctx) findStandard() (Target, bool) {
	t, nearest, ok := pickTarget(c)
	if !ok || nearest >= c.B.Profile.Aggro() {
		return Target{}, false
	}

	c.Dist, c.InRange = nearest, c.W.InRange(c.B, t, nearest)

	return t, true
}
