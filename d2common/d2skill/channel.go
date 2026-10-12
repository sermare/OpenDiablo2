package d2skill

// Sustained casts (the channel mechanism, wave 24). Inferno is cast by holding the button: the client keeps
// asking for the same skill and the server decides, from a channel kept on the caster, whether a run is the
// wind-up or a real flame. VERIFIED in Game.exe 1.14b:
//
//   - Start function SRVST_011_Inferno (0x5c7050): unless the caster has state 12 the cast needs mana >= startmana
//     (shifted 8); the skill needs a valid srvmissilea; then SRVST_InfernoSetupChannel (0x5c6ef0) runs.
//   - SetupChannel: it stores a deadline, now + max(1, calc2), in the skill's target-type field (the slot Frenzy
//     uses as well). With no state-12 list on the caster, it creates one that ends 20 frames later, switches
//     state 12 on, sets the field to 0 and returns (the first run of a hold is a wind-up). With the list present
//     it moves the end to now + 6, shifts the pending action events by one frame, and creates the flame.
//   - Do function SRVDO_019 -> SRVDO_InfernoSpawnFlameMissile (0x5c6d40): a flame is made only while the field is
//     not 0; afterwards the field becomes 1, state 12 goes off and the end of the action is scheduled
//     monstats2 InfernoLen frames later (VERIFIED, SRVDO_InfernoScheduleAnimEnd 0x5c6cb0: the unit's class number
//     indexes monstats, that row's monstats2 link gives the byte; 1 frame when there is no row). The port does
//     not use the byte: the host adaptation below stands in for it.
//
// The engine's hero cast animation is not shortened like the exe's loop, so Do refreshes the channel window by
// ChannelAnimSlack frames (a host adaptation, UNVERIFIED) to keep a held button from losing the channel between
// two animations.

// Channel start function id (skills.txt srvstfunc) and the exe's windows in frames.
const (
	stChannel          = 11
	ChannelStartFrames = 20 // life of the state-12 list made by the first run
	ChannelHoldFrames  = 6  // the end it is moved to by every run while the list exists
)

// channel is the per caster state of a sustained cast.
type channel struct {
	skill    int
	until    int // first frame the state-12 list is gone
	field    int // the skill's target-type field: 0 wind-up, 1 after a flame, else the deadline
	deadline int
	on       bool // unit state 12 is switched on
	// actionEnd is the game frame the exe schedules the end of the action for after a Do (now + InfernoLen).
	actionEnd int
}

// IsChannelStart reports whether the skill's start function is the held-button channel.
func IsChannelStart(sk *Skill) bool { return sk != nil && sk.SrvStFunc == stChannel }

// ChannelOn reports whether the caster has the channel state (unit state 12, the "inferno" row) switched on: from
// the start of a run until its Do.
func (p *Pipeline) ChannelOn(u Unit) bool {
	c := p.channels[u.ID()]

	return c != nil && c.on && c.until > p.frame()
}

// ChannelActive reports whether a state-12 list of the caster is still alive (the next run is a flame).
func (p *Pipeline) ChannelActive(u Unit) bool {
	c := p.channels[u.ID()]

	return c != nil && c.until > p.frame()
}

// channelGate is the mana test of the start function: without the state the caster needs startmana (8.8).
func (p *Pipeline) channelGate(u Unit, sk *Skill) bool {
	return p.ChannelActive(u) || u.Mana() >= sk.StartMana<<8
}

// channelStart is SRVST_InfernoSetupChannel. calc2 is the evaluated calc2 of the skill. It reports whether this
// run is the first of a hold (a wind-up that makes no flame).
func (p *Pipeline) channelStart(u Unit, sk *Skill, calc2 int) (first bool) {
	if p.channels == nil {
		p.channels = map[string]*channel{}
	}

	now := p.frame()
	c := p.channels[u.ID()]

	if c == nil || c.skill != sk.ID {
		c = &channel{skill: sk.ID}
		p.channels[u.ID()] = c
	}

	if calc2 < 1 {
		calc2 = 1
	}

	c.deadline = now + calc2

	if c.until <= now { // no live list: the first run
		c.until, c.field, c.on = now+ChannelStartFrames, 0, true

		return true
	}

	c.until, c.field, c.on = now+ChannelHoldFrames, c.deadline, true

	return false
}

// channelDo is the gate of SRVDO_InfernoSpawnFlameMissile: it reports whether this Do makes a flame, then sets
// the field to 1 and switches the state off.
func (p *Pipeline) channelDo(u Unit) bool {
	c := p.channels[u.ID()]
	if c == nil {
		return true // no start ran (a monster, a scripted cast): the plain one flame per run
	}

	spawn := c.field != 0
	c.field, c.on = 1, false

	// HOST ADAPTATION (UNVERIFIED, w42): in the exe the held button keeps asking while the animation runs, so every
	// request lands inside the 6 frame window of the one before. The port asks again only when the action has
	// ended (InfernoLen frames after this Do), which came 8 frames after the previous Start and made every second
	// run a wind-up (found by scenario w42-inferno-hold). The Do therefore keeps the list alive for the window.
	if now := p.frame(); c.until < now+ChannelHoldFrames {
		c.until = now + ChannelHoldFrames
	}

	c.actionEnd = p.frame() + p.infernoLen(u)

	return spawn
}

// infernoLen is SRVDO_InfernoScheduleAnimEnd's delay (0x5c6cb0, VERIFIED w39): the unit's class number indexes
// monstats, the row's monstats2 link gives the InfernoLen byte; 1 frame when no row is found. (For the seven hero
// classes the owner's tables hold 1.)
func (p *Pipeline) infernoLen(u Unit) int {
	if p.Opt.InfernoLenOf != nil {
		if n, ok := p.Opt.InfernoLenOf(u); ok && n >= 0 {
			return n
		}
	}

	return 1
}

// ChannelActionEnd is the game frame at which the exe ends the action of the last Inferno Do of u (0 before any).
func (p *Pipeline) ChannelActionEnd(u Unit) int {
	if c := p.channels[u.ID()]; c != nil {
		return c.actionEnd
	}

	return 0
}

// ChannelBridge keeps the channel list of u alive across a render stall (HOST ADAPTATION, UNVERIFIED, w47): in the
// exe the held button asks again on every game frame, so a request always lands inside the window. The port asks
// once per rendered frame, and on a loaded machine one frame can cover more game ticks than the window is long
// (found by scenario w42-inferno-hold under load: a second wind-up in the middle of a hold). A list that ended
// after `since` (the first tick of the render frame the request belongs to) did so inside that one frame, so the
// request would have reached it in the exe: the list lives on. It reports whether it bridged.
func (p *Pipeline) ChannelBridge(u Unit, since int) bool {
	c := p.channels[u.ID()]
	now := p.frame()

	if c == nil || c.until > now || c.until <= since {
		return false
	}

	c.until = now + 1

	return true
}
