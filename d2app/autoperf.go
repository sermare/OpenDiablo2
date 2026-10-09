package d2app

import (
	"os"
	"os/signal"
	"runtime"
	"runtime/pprof"
	"sort"
	"strconv"
	"syscall"
	"time"
)

const (
	autoPerfDefaultSeconds = 20.0
	autoPerfDefaultWarmup  = 6.0
	nsPerMs                = 1e6
	cpuProfileHz           = 500
	slowPassMs             = 40.0 // passes slower than this are logged (first maxSlowLogged only)
	maxSlowLogged          = 12
)

// autoPerf implements the frame-time meter and the pprof hooks used to profile
// the engine in the headless-launch scenarios:
//
//	OD2_AUTOPERF=1             meter every frame (update = advance pass, render = draw pass,
//	                           frame = interval between two consecutive draws) and exit when done
//	OD2_AUTOPERF_SECONDS=N     measured seconds (default 20)
//	OD2_AUTOPERF_WARMUP=N      seconds ignored first, covers level loading (default 6)
//	OD2_PPROF_CPU=<file>       write a CPU profile of the whole run
//	OD2_PPROF_HEAP=<file>      write a heap profile at the end
//
// Output: `PERF <kind> frames=.. avg_ms=.. p95_ms=.. p99_ms=.. max_ms=..` for kind update, render
// and frame, then `PERF runtime ...` with goroutine/heap/GC numbers. The meter costs two
// clock reads per pass; it is a no-op when the env vars are unset.
type autoPerf struct {
	enabled bool
	seconds float64
	warmup  float64

	start      time.Time
	lastDraw   time.Time
	update     []float64
	render     []float64
	frame      []float64
	gcStart    runtime.MemStats
	gcCaptured bool
	cpuStart   time.Duration
	slowLogged int

	cpuFile  *os.File
	heapPath string
	done     bool
}

func newAutoPerf() *autoPerf {
	p := &autoPerf{
		seconds:  autoPerfDefaultSeconds,
		warmup:   autoPerfDefaultWarmup,
		heapPath: os.Getenv("OD2_PPROF_HEAP"),
	}

	p.enabled = os.Getenv("OD2_AUTOPERF") != "" && os.Getenv("OD2_AUTOPERF") != "0"

	if v, err := strconv.ParseFloat(os.Getenv("OD2_AUTOPERF_SECONDS"), 64); err == nil && v > 0 {
		p.seconds = v
	}

	if v, err := strconv.ParseFloat(os.Getenv("OD2_AUTOPERF_WARMUP"), 64); err == nil && v >= 0 {
		p.warmup = v
	}

	if p.enabled {
		n := int((p.seconds + 5) * 200)
		p.update = make([]float64, 0, n)
		p.render = make([]float64, 0, n)
		p.frame = make([]float64, 0, n)
	}

	return p
}

// measuring reports whether the warm-up period is over (and starts its clock on first use).
func (p *autoPerf) measuring() bool {
	if p == nil || !p.enabled || p.done {
		return false
	}

	if p.start.IsZero() {
		p.start = time.Now()
	}

	return time.Since(p.start).Seconds() >= p.warmup
}

// startCPUProfile starts the OD2_PPROF_CPU profile. With the meter on this happens when the
// warm-up ends, so the profile shows the steady state and not level loading.
func (a *App) startCPUProfile() {
	p := a.perf
	if path := os.Getenv("OD2_PPROF_CPU"); path != "" && p.cpuFile == nil {
		f, err := os.Create(path)
		if err == nil {
			runtime.SetCPUProfileRate(cpuProfileHz)
			err = pprof.StartCPUProfile(f)
		}

		if err != nil {
			a.Errorf("OD2_PPROF_CPU: %v", err)
		} else {
			p.cpuFile = f
		}
	}
}

func (a *App) startPprofEnv() {
	p := a.perf
	if !p.enabled {
		a.startCPUProfile()
	}

	if os.Getenv("OD2_PPROF_CPU") != "" || p.heapPath != "" {
		// make sure a plain kill still writes the profiles
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-ch
			a.stopPprofEnv()
			os.Exit(0)
		}()
	}
}

func (a *App) stopPprofEnv() {
	p := a.perf
	if p == nil {
		return
	}

	if p.cpuFile != nil {
		pprof.StopCPUProfile()
		_ = p.cpuFile.Close()
		p.cpuFile = nil

		a.Infof("PERF cpu profile written")
	}

	if p.heapPath != "" {
		if f, err := os.Create(p.heapPath); err == nil {
			runtime.GC()
			_ = pprof.Lookup("allocs").WriteTo(f, 0)
			_ = f.Close()

			a.Infof("PERF heap profile written to %s", p.heapPath)
		}

		p.heapPath = ""
	}
}

// perfStamp returns the start time of a pass, or the zero time while not measuring.
func (a *App) perfStamp() time.Time {
	if !a.perf.measuring() {
		return time.Time{}
	}

	if !a.perf.gcCaptured {
		runtime.ReadMemStats(&a.perf.gcStart)
		a.perf.gcCaptured = true
		a.perf.cpuStart = processCPU()

		a.startCPUProfile()

		if a.perf.heapPath != "" {
			// allocation baseline: `go tool pprof -base <path>.base -sample_index=alloc_space <bin> <path>`
			if f, err := os.Create(a.perf.heapPath + ".base"); err == nil {
				_ = pprof.Lookup("allocs").WriteTo(f, 0)
				_ = f.Close()
			}
		}
	}

	return time.Now()
}

func (a *App) perfUpdateDone(t0 time.Time) {
	p := a.perf
	ms := float64(time.Since(t0)) / nsPerMs
	p.update = append(p.update, ms)

	if ms >= slowPassMs && p.slowLogged < maxSlowLogged {
		p.slowLogged++

		a.Infof("PERF slow update pass %.1f ms at t=%.2fs", ms, time.Since(p.start).Seconds())
	}
}

func (a *App) perfRenderDone(t0 time.Time) {
	p := a.perf
	now := time.Now()
	ms := float64(now.Sub(t0)) / nsPerMs
	p.render = append(p.render, ms)

	if ms >= slowPassMs && p.slowLogged < maxSlowLogged {
		p.slowLogged++

		a.Infof("PERF slow render pass %.1f ms at t=%.2fs", ms, now.Sub(p.start).Seconds())
	}

	if !p.lastDraw.IsZero() {
		p.frame = append(p.frame, float64(t0.Sub(p.lastDraw))/nsPerMs)
	}

	p.lastDraw = t0

	if now.Sub(p.start).Seconds() >= p.warmup+p.seconds {
		a.perfFinish()
	}
}

// processCPU is the user+system CPU time the process has used so far; unlike wall-clock frame
// times it is not inflated when other programs compete for the machine.
func processCPU() time.Duration {
	var ru syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &ru) != nil {
		return 0
	}

	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano())
}

func perfStats(v []float64) (avg, p95, p99, mx float64) {
	if len(v) == 0 {
		return
	}

	s := append([]float64(nil), v...)
	sort.Float64s(s)

	sum := 0.0
	for _, x := range s {
		sum += x
	}

	at := func(q float64) float64 { return s[int(float64(len(s)-1)*q)] }

	return sum / float64(len(s)), at(0.95), at(0.99), s[len(s)-1]
}

func (a *App) perfFinish() {
	p := a.perf
	p.done = true

	for _, k := range []struct {
		name string
		v    []float64
	}{{"update", p.update}, {"render", p.render}, {"frame", p.frame}} {
		avg, p95, p99, mx := perfStats(k.v)
		a.Infof("PERF %s frames=%d avg_ms=%.3f p95_ms=%.3f p99_ms=%.3f max_ms=%.3f", k.name, len(k.v), avg, p95, p99, mx)
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	a.Infof("PERF runtime goroutines=%d heap_alloc_mb=%.1f heap_sys_mb=%.1f gc_cycles=%d gc_pause_total_ms=%.2f "+
		"alloc_mb_per_s=%.1f mallocs_per_frame=%.0f",
		runtime.NumGoroutine(), float64(m.HeapAlloc)/bytesToMegabyte, float64(m.HeapSys)/bytesToMegabyte,
		m.NumGC-p.gcStart.NumGC, float64(m.PauseTotalNs-p.gcStart.PauseTotalNs)/nsPerMs,
		float64(m.TotalAlloc-p.gcStart.TotalAlloc)/bytesToMegabyte/p.seconds,
		float64(m.Mallocs-p.gcStart.Mallocs)/float64(len(p.render)+1))

	a.Infof("PERF cpu frames=%d cpu_ms_per_frame=%.3f", len(p.render),
		float64(processCPU()-p.cpuStart)/nsPerMs/float64(len(p.render)+1))

	a.stopPprofEnv()
	os.Exit(0)
}
