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

// Apply puts a state on the unit. A state with the same name is replaced (the
// game frees the old statlist when the skill or level differs and refreshes
// it otherwise). With defs, active states of the same nonzero group, and
// other curses when a curse is applied, end first (states.txt group / curse;
// U: exe confirmation). The previous instance of the same name is returned
// (nil if none was active).
func (s *Set) Apply(frame int, in Instance) *Instance {
	prev := s.states[in.Name]
	if !prev.Active(frame) {
		prev = nil
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

	return prev
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

// AddStream starts a poison or burn stream. Streams are independent and add
// up (U: whether the game stacks or replaces them).
func (s *Set) AddStream(frame int, kind string, perFrame, frames int, source string, skillID int) {
	if perFrame <= 0 || frames <= 0 {
		return
	}

	s.streams = append(s.streams, Stream{Kind: kind, PerFrame: perFrame, Until: frame + frames, Source: source, SkillID: skillID})
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
// streams. Without defs everything is cleared (same as Reset).
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

// ColorShift returns the PL2 hue variation the unit is drawn with: among
// the active states with a colour, the blue ones first, then the highest
// colorpri, then the lowest id. ok is false when no state colours the unit.
func (s *Set) ColorShift(frame int) (shift int, ok bool) {
	var best Def

	for n, in := range s.states {
		d, has := s.defs[n]
		if !has || d.ColorPri == 0 || !in.Active(frame) {
			continue
		}

		if !ok || beats(d, best) {
			best, ok = d, true
		}
	}

	return best.ColorShift, ok
}

func beats(a, b Def) bool {
	if a.Blue != b.Blue {
		return a.Blue
	}

	if a.ColorPri != b.ColorPri {
		return a.ColorPri > b.ColorPri
	}

	return a.ID < b.ID
}

// Reset clears everything (death, new game).
func (s *Set) Reset() {
	s.states = map[string]*Instance{}
	s.streams = nil
	s.frac = map[string]int{}
}
