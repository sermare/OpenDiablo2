package d2monster

import "strings"

// Target modes of the AI table (field +0 of the 16-byte entries).
const (
	TargetNone     = 0 // do not acquire a target (NPCs, sentries)
	TargetStandard = 1 // acquire the nearest hostile or sleep (VERIFIED)
	TargetOnly     = 2 // acquire only: the think function runs without a target if none
)

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

	// 3. MONAI_PostTargetChecks (Summoner wake-up delay, wounded MonTeleport,
	// threat re-targeting) is NOT ported: only used by AIs outside this port.

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
	switch c.B.Def.TargetMode {
	case TargetNone:
		return true
	case TargetStandard, TargetOnly:
	default:
		return true
	}

	t, nearest, ok := pickTarget(c)

	thr := c.B.Profile.Aggro()
	if ok && nearest <= thr { // "<=" vs "<" at the threshold is UNVERIFIED
		c.Target = &t
		c.B.TargetID, c.B.HasTarget = t.ID, true
		c.Dist = nearest
		c.InRange = c.W.InRange(c.B, t, nearest)

		return true
	}

	c.B.HasTarget = false

	if c.B.Def.TargetMode == TargetOnly {
		return true
	}

	// MONAI_AcquireStandardTargetOrSleep (VERIFIED): nothing to hunt, sleep
	// 25 if the nearest player is farther than 34, else (dist-10) if more
	// than 24, else 10. Which distance is used when no player is within the
	// aggro radius is UNVERIFIED: the nearest one regardless of radius.
	// The aggressive-flag wander(5) branch is not ported (flag meaning
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
