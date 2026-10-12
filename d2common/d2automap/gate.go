package d2automap

// UpdateDelayFrames is what AUTOMAP_SetUpdateDelay2 0x4547e0 stores in DAT_0079d22c (VERIFIED, w44): after a level
// change the next two reveal calls (AUTOMAP_UpdateRevealForPlayerMove 0x4546b0, one per frame) only count the
// delay down and scan nothing, then the normal movement test resumes. (That the level change is the caller is
// inferred; the function has a single caller that was not traced.)
const UpdateDelayFrames = 2

// RevealGate is that frame delay.
type RevealGate struct{ delay int }

// Arm starts the delay (a new level was entered).
func (g *RevealGate) Arm() { g.delay = UpdateDelayFrames }

// Ready is called once per frame; it reports whether the reveal may run this frame.
func (g *RevealGate) Ready() bool {
	if g.delay > 0 {
		g.delay--

		return false
	}

	return true
}
