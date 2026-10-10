package d2util

import (
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// OD2_TURBO=1 is the fast mode of scripted runs. The game clock becomes virtual: it moves exactly one
// 25 Hz tick (TurboTickSeconds) per update call and never by wall time, so a run has the same ticks, in
// the same order, however fast the machine is. The renderer then calls the update many times per frame
// until a wall-clock budget is spent, and draws only every Nth frame (or when a capture was requested).
// OD2_TURBO_BUDGET_MS (default 12) and OD2_TURBO_DRAW_EVERY (default 10) tune it.
const (
	// TurboTickSeconds is the game time of one turbo update: one 25 Hz game tick.
	TurboTickSeconds = 1.0 / 25.0
	// TurboMaxTicksPerFrame bounds the updates of one frame.
	TurboMaxTicksPerFrame = 2000

	turboDefaultBudgetMs  = 12.0
	turboDefaultDrawEvery = 10
)

var (
	turboOnce      sync.Once
	turboOn        bool
	turboBudget    = turboDefaultBudgetMs / 1000
	turboDrawEvery = turboDefaultDrawEvery
	turboBase      = float64(time.Now().UnixNano()) / nanoseconds
	turboTicks     atomic.Int64
	turboForceDraw atomic.Bool
)

func turboInit() {
	turboOnce.Do(func() {
		if v := os.Getenv("OD2_TURBO"); v != "" && v != "0" {
			turboOn = true
		}

		if f, err := strconv.ParseFloat(os.Getenv("OD2_TURBO_BUDGET_MS"), 64); err == nil && f > 0 {
			turboBudget = f / 1000
		}

		if n, err := strconv.Atoi(os.Getenv("OD2_TURBO_DRAW_EVERY")); err == nil && n >= 1 {
			turboDrawEvery = n
		}
	})
}

// TurboEnabled reports OD2_TURBO.
func TurboEnabled() bool { turboInit(); return turboOn }

// TurboBudgetSeconds is the wall-clock span of update calls per frame.
func TurboBudgetSeconds() float64 { turboInit(); return turboBudget }

// TurboDrawEvery is N of "draw every Nth frame".
func TurboDrawEvery() int { turboInit(); return turboDrawEvery }

// TurboAdvance moves the virtual clock by one tick.
func TurboAdvance() { turboTicks.Add(1) }

// TurboTicks is the number of virtual ticks so far.
func TurboTicks() int64 { return turboTicks.Load() }

// TurboRequestDraw makes the renderer draw the current frame (a capture was requested).
func TurboRequestDraw() { turboForceDraw.Store(true) }

// TurboTakeDraw reports and clears a draw request.
func TurboTakeDraw() bool { return turboForceDraw.Swap(false) }

// TurboPeekDraw reports a draw request without clearing it.
func TurboPeekDraw() bool { return turboForceDraw.Load() }

// TurboContinue is the tick budget policy: whether a frame runs another update after `ticks` updates and
// `spent` wall seconds. A forced draw ends the frame at once so the capture sees the state that asked.
func TurboContinue(ticks int, spent, budget float64, forceDraw bool) bool {
	if forceDraw || ticks >= TurboMaxTicksPerFrame {
		return false
	}

	return ticks < 1 || spent < budget
}

// TurboShouldDraw is the skip-draw policy: frame numbers count from 0; every Nth frame is drawn, and
// any forced one.
func TurboShouldDraw(frame, every int, forced bool) bool {
	return forced || every <= 1 || frame%every == 0
}
