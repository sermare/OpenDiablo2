package d2state

import "sort"

// StatMod is one stat a state gives (ItemStatCost name, signed value).
type StatMod struct {
	Stat  string
	Value int
}

// Instance is one active state on a unit.
type Instance struct {
	Name string
	// Until is the first frame the state is no longer active; 0 means it
	// never ends by itself (toggled auras, shapeshift forms).
	Until int
	Mods  []StatMod
	// Source is the id of the unit that applied it; SkillID and Level are the
	// skill (stats 0x15e and 0x15f in the game).
	Source  string
	SkillID int
	Level   int
	// Count is a small counter for stacking states (assassin charges,
	// Frenzy hits).
	Count int
}

// Active reports whether the instance is in force at a frame.
func (in *Instance) Active(frame int) bool { return in != nil && (in.Until == 0 || in.Until > frame) }

// Stream is a damage over time stream (poison or burn).
type Stream struct {
	Kind     string // "poison" or "burn"
	PerFrame int    // 8.8 hit points per frame
	Until    int
	Source   string
	SkillID  int
}

// Set is the states and DoT streams of one unit.
type Set struct {
	states  map[string]*Instance
	streams []Stream
	frac    map[string]int // 8.8 remainder per stream kind
	defs    Defs           // states.txt rows; nil = no exclusion or death rules
}

// New creates an empty set.
func New() *Set {
	return &Set{states: map[string]*Instance{}, frac: map[string]int{}}
}

// SetDefs gives the set the states.txt rows that drive group / curse
// exclusion, removal on hit and death, and colour. Without them the set only
// replaces a state of the same name.
func (s *Set) SetDefs(d Defs) { s.defs = d }

// Apply puts a state on the unit and returns the active instance it
// replaced (nil if none). It is ApplyTimed without the accepted flag.
func (s *Set) Apply(frame int, in Instance) *Instance {
	_, prev := s.ApplyTimed(frame, in)

	return prev
}

// ApplyTimed follows SKILL_CreateTimedStateStatList (0x56c740, verified):
//
//   - the unit's existing statlist is found by state id, except for curses
//     (states.txt curse = 1) where any active curse is found, whatever its
//     name: a unit carries one curse at a time;
//   - a state with the same name from the same skill (SkillID != 0 on both)
//     and the same level only has its end refreshed (to now + length, even
//     when that is sooner; a length of 0 changes nothing) and is not rebuilt;
//   - the same skill at a lower level than the active one is rejected;
//   - anything else (higher level, other skill, other curse) removes the old
//     statlist and creates the new one.
//
// Without a SkillID (stun, chill, auras, tests) a state of the same name is
// simply replaced. The States.txt group column is NOT read here (verified):
// group exclusion is ClearGroup, called by the cast code. Shrine states are
// not curses (U). The returned prev is the replaced or refreshed instance.
func (s *Set) ApplyTimed(frame int, in Instance) (applied bool, prev *Instance) {
	var old string

	if d, ok := s.defs[in.Name]; ok && d.Curse && !isShrine(in.Name) {
		for n, x := range s.states {
			if dx, ok := s.defs[n]; ok && dx.Curse && !isShrine(n) && x.Active(frame) {
				old = n
			}
		}
	}

	if old == "" {
		old = in.Name
	}

	if ex := s.states[old]; ex.Active(frame) {
		prev = ex

		if old == in.Name && in.SkillID != 0 && ex.SkillID == in.SkillID {
			if in.Level == ex.Level {
				if in.Until != 0 {
					ex.Until = in.Until
				}

				return true, prev
			}

			if in.Level < ex.Level {
				return false, prev
			}
		}

		delete(s.states, old)
	}

	if s.defs != nil {
		for n := range s.states {
			if s.defs.exclusive(in.Name, n) {
				delete(s.states, n)
			}
		}
	}

	cp := in
	cp.Mods = append([]StatMod(nil), in.Mods...)
	s.states[in.Name] = &cp

	return true, prev
}

// ClearGroup is FUN_0056a480 called with its flag set (verified, 0x56a480):
// for a state whose States.txt group is nonzero it ends every active state of
// that group, the state itself included (toggles the bit off and frees its
// statlist); group 0 does nothing. Only the buff do-function SRVDO_
// FrozenArmorState (0x5c7540, srvdofunc 18), BladeShield, Whirlwind, Wearwolf
// and a few more call it, right before the timed statlist is created, so
// Frozen / Shiver / Chilling / Bone Armor (group 1, with justhit) replace each
// other and Burst of Speed (quickness) and Fade (group 2) do the same.
// SKILL_CreateTimedStateStatList (0x56c740) itself never reads the column.
// It reports whether any state was ended.
func (s *Set) ClearGroup(frame int, name string) bool {
	d, ok := s.defs[name]
	if !ok || d.Group == 0 {
		return false
	}

	ended := false

	for n, x := range s.states {
		if dx, ok := s.defs[n]; ok && dx.Group == d.Group && x.Active(frame) {
			delete(s.states, n)

			ended = true
		}
	}

	return ended
}

// Get returns the active instance of a state or nil.
func (s *Set) Get(frame int, name string) *Instance {
	if in := s.states[name]; in.Active(frame) {
		return in
	}

	return nil
}

// Active reports whether a state is in force.
func (s *Set) Active(frame int, name string) bool { return s.Get(frame, name) != nil }

// ActiveID reports whether the state with the states.txt row number id is in
// force at a frame (false without the table).
func (s *Set) ActiveID(frame, id int) bool {
	for _, d := range s.defs {
		if d.ID == id {
			return s.Active(frame, d.Name)
		}
	}

	return false
}

// Remove ends a state at once.
func (s *Set) Remove(name string) { delete(s.states, name) }

// Names lists the active states, sorted.
func (s *Set) Names(frame int) []string {
	var out []string

	for n, in := range s.states {
		if in.Active(frame) {
			out = append(out, n)
		}
	}

	sort.Strings(out)

	return out
}

// Stat sums a stat over all active states.
func (s *Set) Stat(frame int, stat string) int {
	total := 0

	for _, in := range s.states {
		if !in.Active(frame) {
			continue
		}

		for _, m := range in.Mods {
			if m.Stat == stat {
				total += m.Value
			}
		}
	}

	return total
}

// StatMods sums every stat over all active states (what Stat does for one
// name). Zero totals are left out.
func (s *Set) StatMods(frame int) map[string]int {
	out := map[string]int{}

	for _, in := range s.states {
		if !in.Active(frame) {
			continue
		}

		for _, m := range in.Mods {
			out[m.Stat] += m.Value
		}
	}

	for k, v := range out {
		if v == 0 {
			delete(out, k)
		}
	}

	return out
}

// AddStream starts a poison or burn stream. Verified (0x578990 poison,
// 0x578b00 burn): a unit has one statlist per kind (state 2 and state 0x73,
// both carrying hpregen 0x4a = -perFrame). A new stream replaces the active
// one (new strength and new end) only when its per-frame damage is at least
// the active one's; a weaker one is ignored entirely. Poison and burn are
// independent and add up. A value or length of 0 does nothing.
func (s *Set) AddStream(frame int, kind string, perFrame, frames int, source string, skillID int) {
	if perFrame <= 0 || frames <= 0 {
		return
	}

	st := Stream{Kind: kind, PerFrame: perFrame, Until: frame + frames, Source: source, SkillID: skillID}

	for i := range s.streams {
		cur := &s.streams[i]
		if cur.Kind != kind || cur.Until <= frame {
			continue
		}

		if perFrame >= cur.PerFrame {
			*cur = st
		}

		return
	}

	s.streams = append(s.streams, st)
}

// Streams returns the active streams.
func (s *Set) Streams(frame int) []Stream {
	var out []Stream

	for _, st := range s.streams {
		if st.Until > frame {
			out = append(out, st)
		}
	}

	return out
}

// TickResult is what one frame of Tick produced.
type TickResult struct {
	// Poison and Burn are the whole hit points lost this frame.
	Poison, Burn int
	// Expired lists the states that ended on this frame.
	Expired []string
}

// Tick advances the set by one frame: it sums the DoT streams active on
// frame, carries the fractional part to the next call and drops ended states
// and streams. Call it once per frame with increasing frame numbers.
func (s *Set) Tick(frame int) TickResult {
	var res TickResult

	var perKind = map[string]int{}

	live := s.streams[:0]

	for _, st := range s.streams {
		if st.Until <= frame {
			continue
		}

		perKind[st.Kind] += st.PerFrame

		live = append(live, st)
	}

	s.streams = live

	for kind, v := range perKind {
		total := s.frac[kind] + v
		whole := total >> 8
		s.frac[kind] = total & 0xff

		switch kind {
		case "poison":
			res.Poison += whole
		default:
			res.Burn += whole
		}
	}

	var names []string

	for n, in := range s.states {
		if in.Until != 0 && in.Until <= frame {
			names = append(names, n)
		}
	}

	sort.Strings(names)

	for _, n := range names {
		delete(s.states, n)
	}

	res.Expired = names

	return res
}

// Drain takes up to amount from the positive pool a stat holds (Bone Armor's
// bonearmor, in the stat's own units), oldest state name first, and returns
// how much was taken. A state whose pool reaches zero ends.
func (s *Set) Drain(frame int, stat string, amount int) int {
	taken := 0

	for _, n := range s.Names(frame) {
		in := s.states[n]
		hit := false

		for i := range in.Mods {
			if in.Mods[i].Stat != stat || in.Mods[i].Value <= 0 || taken >= amount {
				continue
			}

			hit = true
			x := amount - taken

			if x > in.Mods[i].Value {
				x = in.Mods[i].Value
			}

			in.Mods[i].Value -= x
			taken += x
		}

		if !hit {
			continue
		}

		empty := true

		for _, m := range in.Mods {
			if m.Stat == stat && m.Value > 0 {
				empty = false
			}
		}

		if empty {
			delete(s.states, n)
		}
	}

	return taken
}

// Hit ends the states flagged remhit (cloak_of_shadows) and returns their
// names; call it when the unit takes a hit. Needs defs.
func (s *Set) Hit(frame int) []string {
	var out []string

	for n, in := range s.states {
		if d, ok := s.defs[n]; ok && d.RemHit && in.Active(frame) {
			out = append(out, n)
		}
	}

	sort.Strings(out)

	for _, n := range out {
		delete(s.states, n)
	}

	return out
}

// Death clears what a dying unit of a kind ("player", "monster", "boss")
// loses: every state without the matching *staydeath flag, and all DoT
// streams. Verified (STATS_RemoveNonPersistentStatesOnDeath, flags test
// 0x63b5b0): the statlists use plrstaydeath for players and monstaydeath for
// every monster, bosses included; bossstaydeath only picks which state bits
// (visuals) survive, see Defs.BitStays. Without defs everything is cleared
// (same as Reset).
func (s *Set) Death(kind string) {
	for n := range s.states {
		if !s.defs.stays(n, kind) {
			delete(s.states, n)
		}
	}

	s.streams = nil
	s.frac = map[string]int{}
}

// Shatters reports whether the unit would shatter if it died now (an active
// state with the shatter flag: freeze).
func (s *Set) Shatters(frame int) bool {
	for n, in := range s.states {
		if d, ok := s.defs[n]; ok && d.Shatter && in.Active(frame) {
			return true
		}
	}

	return false
}

// ColorShift returns the PL2 hue variation the unit is drawn with: among the
// active states with colorpri > 0 and id > 0, the highest colorpri, ties to
// the lowest id (verified, client FUN_004d65a0). The "blue" column plays no
// part. ok is false when no state colours the unit.
func (s *Set) ColorShift(frame int) (shift int, ok bool) {
	var best Def

	for n, in := range s.states {
		d, has := s.defs[n]
		if !has || d.ColorPri <= 0 || d.ID <= 0 || !in.Active(frame) {
			continue
		}

		if !ok || d.ColorPri > best.ColorPri || (d.ColorPri == best.ColorPri && d.ID < best.ID) {
			best, ok = d, true
		}
	}

	return best.ColorShift, ok
}

// ColorShiftLocal is ColorShift for the local player: a shift of 104 (the
// poison green) is not applied to the player's own sprite when the video
// mode is 3D or higher (verified, FUN_004d65a0: shift 0 is stored instead).
func (s *Set) ColorShiftLocal(frame int, mode3D bool) (shift int, ok bool) {
	shift, ok = s.ColorShift(frame)
	if ok && shift == 104 && mode3D {
		return 0, ok
	}

	return shift, ok
}

// Reset clears everything (death, new game).
func (s *Set) Reset() {
	s.states = map[string]*Instance{}
	s.streams = nil
	s.frac = map[string]int{}
}
