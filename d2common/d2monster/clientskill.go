package d2monster

// Animation driven client side monster skills (cltdofunc), read from
// Game.exe 1.14b (SkillsMon.cpp / SkillsEMon.cpp, notes in gaps-slice-F.md):
// Diablo Run (CLTDO_059 0x4dedd0), Mosquito (062, 0x4df3b0), Queen death (064,
// 0x4df5c0) and the Siege Beast stomp shake (075, 0x4eda40). They are called on
// animation frames and edit the unit's animation state; here they are pure
// functions from (frame, counters) to a FrameEdit.

// Client do-function numbers.
const (
	CltDoDiabRun    = 59
	CltDoMosquito   = 62
	CltDoQueenDeath = 64
	CltDoStomp      = 75
)

// FrameEdit is what a client skill function asks the animation to do. Totals
// are in the exe's 24.8 fixed point (frames << 8).
type FrameEdit struct {
	Handled   bool // the function returned 1
	SetSeq    bool
	Seq       int // sequence frame to jump to
	SetTotal  bool
	Total     int  // dwTotalFrames
	TotalLow  bool // only the low byte of dwTotalFrames is replaced by Total (Mosquito)
	StartRun  bool // velocity MulDiv(calc1, stat 0x43, 100), path type 1, recompute the path
	ResetMode bool // reset to mode 0xc and clear flag 0x40 (Queen death end)
	Flag10000 bool // set unit flag 0x10000 (Queen death end)
}

// RunStage values kept in the skill's ActionStage slot.
const (
	runStageIdle    = 0
	runStageRunning = 1
	runStageRestart = 2 // bit 2: restart the animation loop
)

// DiabRunClient is CLTDO_059_DiabRun for one animation event. frame is the
// current animation frame, stage the skill's ActionStage slot, targetNear whether
// the path target unit exists and is within table distance 1. It returns the
// edit and the new stage. VERIFIED from the decompile except the meaning of the
// stage bit 2, which nothing in the studied code sets.
//
//   - frame past Param4: not handled (the function returns 0);
//   - target unit within table distance 1: the run stops: stage 0,
//     total Param1<<8, sequence frame Param2;
//   - stage bit 2 set: same restart;
//   - frame == Param3 and not restarting: start moving, stage 1;
//   - frame == Param4: total Param5<<8, sequence frame Param6 (loop the run cycle).
func DiabRunClient(p [7]int, frame, stage int, targetNear bool) (FrameEdit, int) {
	if frame > p[4] {
		return FrameEdit{}, stage
	}

	restart := FrameEdit{Handled: true, SetSeq: true, Seq: p[2], SetTotal: true, Total: p[1] << 8}

	if targetNear {
		return restart, runStageIdle
	}

	if stage&runStageRestart != 0 {
		return restart, runStageIdle
	}

	if frame == p[3] {
		return FrameEdit{Handled: true, StartRun: true}, runStageRunning
	}

	if frame != p[4] {
		return FrameEdit{Handled: true}, stage
	}

	return FrameEdit{Handled: true, SetSeq: true, Seq: p[6], SetTotal: true, Total: p[5] << 8}, stage
}

// MosquitoClient is CLTDO_062_Mosquito for one event: the bite loops while the
// target (stored in the skill slots) stays in melee range and the counter in
// TargetType has more than one bite left. On a loop the animation restarts at
// frame Param1 and the total's low byte becomes 0x100. Returns the edit and the
// new counter; Handled false ends the skill.
func MosquitoClient(param1, counter int, hasTarget, inMelee bool) (FrameEdit, int) {
	if !hasTarget {
		return FrameEdit{}, 0
	}

	if !inMelee || counter-1 <= 0 {
		return FrameEdit{}, counter
	}

	return FrameEdit{Handled: true, SetSeq: true, Seq: param1, SetTotal: true, Total: 0x100, TotalLow: true}, counter - 1
}

// QueenDeathClient is CLTDO_064_QueenDeath, keyed on the current animation frame
// (dwCurFrame >> 8) with a loop counter in TargetType: frame 6 jumps to sequence
// frame 7 at total 0x3200; frame 0x16 counts a pass and after more than three
// passes jumps to sequence frame 0x28 at total 0x1100; frame 0x1e resets to mode
// 0xc and sets flag 0x10000. Always handled.
func QueenDeathClient(frame, counter int) (FrameEdit, int) {
	e := FrameEdit{Handled: true}

	switch frame {
	case 6:
		e.SetSeq, e.Seq, e.SetTotal, e.Total = true, 7, true, 0x3200
	case 0x16:
		if counter > 3 {
			e.SetSeq, e.Seq, e.SetTotal, e.Total = true, 0x28, true, 0x1100
		} else {
			counter++
		}
	case 0x1e:
		e.ResetMode, e.Flag10000 = true, true
	}

	return e, counter
}

// Shake is a screen shake (GFX_StartScreenShake 0x4727c0 / GFX_UpdateScreenShake
// 0x472a80, VERIFIED): the offset magnitude ramps up linearly over RampMs from
// zero to Amp pixels, holds for HoldMs, then decays linearly over DecayMs. While
// it runs, the view is displaced by a random value in [-m, m] on each axis every
// frame. HoldMs of 0 disables the shake (the exe ignores such a call).
type Shake struct {
	Amp, RampMs, HoldMs, DecayMs int
}

// StompShake builds the Siege Beast stomp shake from skills.txt Param1..4
// (CLTDO_075): amplitude Param1, ramp Param2*40 ms, hold Param3*40 ms, decay
// Param4*40 ms (40 ms = one 25 fps frame). The studied-batch-10 note that read
// Param3/Param4 as the first two arguments was wrong: the call is
// (Param1, Param2*40, Param3*40, Param4*40).
func StompShake(p1, p2, p3, p4 int) Shake {
	return Shake{Amp: p1, RampMs: p2 * 40, HoldMs: p3 * 40, DecayMs: p4 * 40}
}

// Active says whether the shake still runs elapsedMs after it started.
func (s Shake) Active(elapsedMs int) bool {
	return s.HoldMs != 0 && elapsedMs >= 0 && elapsedMs <= s.RampMs+s.HoldMs+s.DecayMs
}

// Magnitude is the maximum displacement in pixels elapsedMs after the start.
func (s Shake) Magnitude(elapsedMs int) int {
	if !s.Active(elapsedMs) {
		return 0
	}

	switch {
	case elapsedMs < s.RampMs:
		return s.Amp * elapsedMs / s.RampMs
	case elapsedMs < s.RampMs+s.HoldMs || s.DecayMs == 0:
		return s.Amp
	}

	return (s.DecayMs - elapsedMs + s.HoldMs + s.RampMs) * s.Amp / s.DecayMs
}
