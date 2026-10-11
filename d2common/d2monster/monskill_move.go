package d2monster

// Movement skills of monsters that need more than a hit: Jump (srvdofunc 89,
// SRVDO_089_Jump 0x5c9be0) and Diablo Run (srvdofunc 103, SRVDO_103_DiabRun
// 0x5cb680). Notes: d2-re-notes/studied-batch-2.md (SkillMonst.cpp) and
// gaps-slice-C.md. Pure rules only; the Director places the unit.
//
// Skills rows (patch_d2 skills.txt): Jump id 165 (calc1 10); DiabRun id 198
// (calc1 20, Param1..6 = 8, 14, 5, 13, 16, 6).

// Do-function numbers.
const (
	DoJump    = 89
	DoDiabRun = 103
)

// MulDiv is the Win32 MulDiv the exe uses for speeds: a*b/c rounded to the
// nearest integer, halves away from zero; 0 when c is 0.
func MulDiv(a, b, c int) int {
	if c == 0 {
		return 0
	}

	n := a * b
	if (n < 0) != (c < 0) {
		return (n - c/2) / c
	}

	if c < 0 {
		return (n + c/2) / c
	}

	return (n + c/2) / c
}

// DiabRunSpeed is the velocity of the run (VERIFIED): MulDiv(calc1, moveStat,
// 100), where moveStat is the unit's stat 0x43 (move-speed percent, 100 when
// no override). With calc1 20 and the stat at 100 this is 20.
func DiabRunSpeed(calc1, moveStat int) int { return MulDiv(calc1, moveStat, 100) }

// DiabRunParams are the animation frame numbers the run reads from the skill
// row (VERIFIED roles): the run starts at StartFrame, rewinds from LoopAt to
// LoopTo; Restart* apply when the restart bit (stage bit 2) is set.
type DiabRunParams struct {
	RestartTotal int // Param1: total frames (<<8 in the exe) on restart
	RestartSeq   int // Param2: sequence frame on restart
	StartFrame   int // Param3
	LoopAt       int // Param4
	Total        int // Param5
	LoopTo       int // Param6
}

// DiabRunParamsOf reads the params from skills.txt Param1..Param6.
func DiabRunParamsOf(p [7]int) DiabRunParams {
	return DiabRunParams{RestartTotal: p[1], RestartSeq: p[2], StartFrame: p[3], LoopAt: p[4], Total: p[5], LoopTo: p[6]}
}

// RunStep is what one animation frame of the run asks for.
type RunStep struct {
	Start  bool // begin moving (path type 1, speed DiabRunSpeed)
	Rewind bool // loop the animation back to LoopTo
	Strike bool // roll a hit on the target (action frame fired and in melee reach)
}

// Step advances the run by one frame. frame is the current animation frame,
// started whether the move already began, actionHit the action-frame flag and
// inMelee whether the stored target is within melee range (VERIFIED: the hit
// is rolled once per action frame and the flag cleared).
func (r DiabRunParams) Step(frame int, started, actionHit, inMelee bool) RunStep {
	var s RunStep

	if !started && frame >= r.StartFrame {
		s.Start = true
	}

	if started && r.LoopAt > 0 && frame >= r.LoopAt {
		s.Rewind = true
	}

	if started && actionHit && inMelee {
		s.Strike = true
	}

	return s
}

// RunAdvance is how far (subtiles) the unit closes the gap to a target per
// action frame while running: it stops at melee reach. UNVERIFIED stand-in for
// the exe's per-frame path stepping, sized so the run reaches a hero within a
// few action frames.
func RunAdvance(dist, reach, stride int) int {
	gap := dist - reach
	if gap <= 0 {
		return 0
	}

	if stride > 0 && gap > stride {
		return stride
	}

	return gap
}

// JumpReflect is the point a jumper of class 0x4e heads for after landing
// (VERIFIED arithmetic): 3*dest - 2*victim, i.e. the landing point pushed
// away from the victim. Which monster class 0x4e is stays UNVERIFIED.
func JumpReflect(dest, victim Point) Point {
	return Point{X: 3*dest.X - 2*victim.X, Y: 3*dest.Y - 2*victim.Y}
}

// JumpLanding is where a leaper comes down: on the stored destination, which
// the AI took from the target's position (VERIFIED: the destination is the
// x,y kept with the skill node; the strike resolves on arrival).
func JumpLanding(dest Point) Point { return dest }
