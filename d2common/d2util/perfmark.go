package d2util

import (
	"fmt"
	"sync"
	"time"
)

//nolint:gochecknoglobals // process-wide start-up clock
var (
	perfStart = time.Now()
	perfMu    sync.Mutex
	perfSeen  = map[string]bool{}
	perfLog   = newPerfLogger()
)

func newPerfLogger() *Logger {
	l := NewLogger()
	l.SetPrefix("Perf")

	return l
}

// PerfSince returns the time since the process started (package initialisation).
func PerfSince() time.Duration { return time.Since(perfStart) }

// PerfMark logs `PERF mark <name> t_ms=<ms since process start>` the first time a
// name is marked (later calls with the same name are ignored), so start-up phases can be
// compared between builds. It is cheap enough to leave in production paths.
func PerfMark(name string) {
	perfMu.Lock()
	seen := perfSeen[name]
	perfSeen[name] = true
	perfMu.Unlock()

	if seen {
		return
	}

	perfLog.Info(fmt.Sprintf("PERF mark %s t_ms=%.0f", name, float64(time.Since(perfStart))/float64(time.Millisecond)))
}

// PerfTime logs `PERF span <name> ms=<elapsed>` for a timed section: `defer d2util.PerfTime("x")()`.
func PerfTime(name string) func() {
	t0 := time.Now()

	return func() {
		perfLog.Info(fmt.Sprintf("PERF span %s ms=%.1f", name, float64(time.Since(t0))/float64(time.Millisecond)))
	}
}
