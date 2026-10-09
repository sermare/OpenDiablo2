package d2sfx

import (
	"fmt"
	"math"
	"sort"
)

// TicksPerSecond is the game's logic rate; Fade/Duration columns are in ticks.
const TicksPerSecond = 25

// DefaultVoices is the size of the original voice array (verified: 16 slots of
// 0x20 bytes at 0x7c0b00..0x7c0d00). Whether the original splits them into 2D
// and 3D classes is unverified; this model uses one pool.
const DefaultVoices = 16

const (
	heroPriorityBonus = 0x50 // added as a byte (wraps) when the hero emits (verified)
	maxClamp          = 2000.0
	soloDuckMin       = 70
	soloDuckStep      = 2
	duckMax           = 100
)

// Player is one playable sound (matches d2interface.SoundEffect).
type Player interface {
	Play()
	Stop()
	SetPan(pan float64)
	SetVolume(volume float64)
	IsPlaying() bool
}

// Loader creates a Player for a resolved row. An error marks the play as
// DecisionLoadFailed.
type Loader func(row *Row) (Player, error)

// Clock returns the current game tick.
type Clock interface{ Now() int64 }

// Decision is the outcome of a start attempt.
type Decision int

// Decisions.
const (
	DecisionQueued       Decision = iota // waiting (delay, or no voice yet for a loop)
	DecisionPlayed                       // got a voice and started
	DecisionStolen                       // started by taking a lower-priority voice
	DecisionMerged                       // folded into a recent same-group request (Compound)
	DecisionNoRow                        // index 0 / out of range
	DecisionZeroPriority                 // Priority 0 never plays
	DecisionInaudible                    // beyond the Falloff radius
	DecisionSameTick                     // same sound started within a tick at >= priority
	DecisionDeferred                     // Defer Inst: an instance of the group is already playing
	DecisionNoVoice                      // every voice holds an equal or higher priority sound
	DecisionLoadFailed                   // Loader error
	DecisionFinished                     // ran to completion / Duration elapsed
	DecisionStopped                      // stopped by the caller, stolen from, or replaced (Stop Inst)
)

var decisionNames = [...]string{"queued", "played", "stolen", "merged", "no-row", "zero-priority",
	"inaudible", "same-tick", "deferred", "no-voice", "load-failed", "finished", "stopped"}

func (d Decision) String() string {
	if int(d) < len(decisionNames) {
		return decisionNames[d]
	}

	return fmt.Sprintf("decision(%d)", int(d))
}

type state int

const (
	stQueued state = iota
	stPlaying
	stStopping
	stDead
)

// Request describes one play call.
type Request struct {
	Index        int
	Delay        int64   // ticks before it may start
	HasPos       bool    // positional; otherwise treated as at the hero
	X, Y         float64 // emitter position in the same units as the listener
	Emitter      int     // opaque emitter id, 0 = none (used by Defer/Stop Inst)
	Hero         bool    // emitter is the local hero: priority += 0x50
	FixedVariant bool    // do not pick a Group Size variant (verified flag bit 0)
	NoFade       bool    // skip Fade In (verified flag bit 1)
}

// Report describes a decision, for logs and tests.
type Report struct {
	Requested int
	Picked    int
	Handle    string
	File      string
	Priority  int
	Decision  Decision
	Voice     int // voice slot, -1 if none
	Victim    string
	Dist2     float64
}

func (r Report) String() string {
	return fmt.Sprintf("requested=%d picked=%d handle=%s file=%q priority=%d decision=%s voice=%d victim=%q dist2=%.0f",
		r.Requested, r.Picked, r.Handle, r.File, r.Priority, r.Decision, r.Voice, r.Victim, r.Dist2)
}

type fade struct {
	active         bool
	from, to       int
	startTk, endTk int64
}

// Instance is a queued or playing sound.
type Instance struct {
	req      Request
	seq      int
	index    int // current (possibly picked) row index
	row      *Row
	priority uint8
	st       state
	start    int64
	vol      int // 0..255, starts at 255 (verified)
	dist2    float64
	voice    int
	player   Player
	stopFlag bool // marked to be stopped (verified: Stop Inst sets a flag on the older instance)
	fade     fade
	lastVol  float64
	lastPan  float64
	manPan   *float64 // caller-set pan for non-positional sounds
	report   Report
}

// Report returns the most recent decision for the instance.
func (i *Instance) Report() Report { return i.report }

// Row returns the row currently bound to the instance.
func (i *Instance) Row() *Row { return i.row }

// Playing reports whether the instance holds a voice.
func (i *Instance) Playing() bool { return i.st == stPlaying }

// Alive reports whether the bank still tracks the instance.
func (i *Instance) Alive() bool { return i.st != stDead }

// Volume returns the instance's 0..255 envelope value.
func (i *Instance) Volume() int { return i.vol }

// Bank is a fixed pool of voices plus the pending queue.
type Bank struct {
	table    *Table
	load     Loader
	clock    Clock
	rnd      func(n int) int
	voices   []*Instance
	queue    []*Instance
	history  map[int][2]int // requested row -> last two picks (row +0x64/+0x68)
	seq      int
	lx, ly   float64
	sfxVol   float64
	musicVol float64
	duck     int
	lastTick int64

	// OnReport, if set, receives every notable decision.
	OnReport func(Report)
}

// NewBank makes a bank of n voices (DefaultVoices if n <= 0).
func NewBank(t *Table, n int, load Loader, clock Clock, rnd func(int) int) *Bank {
	if n <= 0 {
		n = DefaultVoices
	}

	return &Bank{table: t, load: load, clock: clock, rnd: rnd, voices: make([]*Instance, n),
		history: map[int][2]int{}, sfxVol: 1, musicVol: 1, duck: duckMax, lastTick: -1}
}

// Table returns the bank's sound table.
func (b *Bank) Table() *Table { return b.table }

// SetListener sets the hero position used for distance and pan.
func (b *Bank) SetListener(x, y float64) { b.lx, b.ly = x, y }

// SetVolumes sets the sfx and music master volumes (0..1).
func (b *Bank) SetVolumes(sfx, music float64) { b.sfxVol, b.musicVol = sfx, music }

// ActiveVoices counts voices in use.
func (b *Bank) ActiveVoices() int {
	n := 0

	for _, v := range b.voices {
		if v != nil {
			n++
		}
	}

	return n
}

// Voice returns the instance occupying a slot (nil if free).
func (b *Bank) Voice(slot int) *Instance { return b.voices[slot] }

// QueueLen returns the number of tracked instances.
func (b *Bank) QueueLen() int { return len(b.queue) }

// PickVariant chooses a member of a Group Size group, avoiding recent picks
// (verified, SOUND_PickGroupVariant): for GroupSize g > 1 the history window is
// min(g-1, 2) entries, shortened by one with probability 1/(g+1) when g <= 3;
// candidates are base+rand(g) and are redrawn while they match the window.
func (b *Bank) PickVariant(base int) int {
	row := b.table.Get(base)
	if row == nil || row.GroupSize <= 1 {
		return base
	}

	g := row.GroupSize

	n := g - 1
	if n > 2 {
		n = 2
	}

	if g <= 3 && b.rnd(g+1) == 0 {
		n--
	}

	h := b.history[base]

	for {
		pick := base + b.rnd(g)
		clash := false

		for k := 0; k < n; k++ {
			if pick == h[k] {
				clash = true
			}
		}

		if !clash {
			return pick
		}
	}
}

// Play queues a sound and runs one update pass, so the returned instance
// already carries a decision (it stays queued if Delay > 0).
func (b *Bank) Play(req Request) *Instance {
	now := b.clock.Now()
	row := b.table.Get(req.Index)

	inst := &Instance{req: req, index: req.Index, row: row, voice: -1, vol: 255}
	inst.report = Report{Requested: req.Index, Picked: req.Index, Voice: -1}

	if row == nil || req.Index <= 0 {
		return b.reject(inst, DecisionNoRow)
	}

	inst.report.Handle, inst.report.File = row.Handle, row.FileName

	if row.Priority == 0 { // verified: priority byte 0 returns before queuing
		return b.reject(inst, DecisionZeroPriority)
	}

	// Compound (shape verified, FUN_004b60a0): reuse a live instance of the same group
	// head started within Compound ticks (negative = any age).
	if row.Compound != 0 {
		head := b.table.Head(req.Index)

		for _, q := range b.queue {
			if q.st != stDead && !q.stopFlag && b.table.Head(q.index) == head &&
				(row.Compound < 0 || now-q.start <= int64(row.Compound)) {
				q.report.Decision = DecisionMerged
				b.emit(q.report)

				return q
			}
		}
	}

	b.seq++
	inst.seq = b.seq
	inst.priority = uint8(row.Priority)

	if req.Hero {
		inst.priority += heroPriorityBonus // byte wrap is intentional
	}

	inst.start = now + req.Delay
	inst.dist2 = b.distance2(req)
	inst.report.Priority = int(inst.priority)
	inst.report.Decision = DecisionQueued
	b.queue = append(b.queue, inst)

	b.process(now)

	return inst
}

// Stop ends an instance; with a Fade Out column it fades first (unverified:
// the fade-out hook was not located in the binary).
func (b *Bank) Stop(inst *Instance) {
	if inst == nil || inst.st == stDead {
		return
	}

	if inst.st == stPlaying && inst.row.FadeOut > 0 {
		now := b.clock.Now()
		inst.fade = fade{active: true, from: inst.vol, to: 0, startTk: now, endTk: now + int64(inst.row.FadeOut)}
		inst.stopFlag = true // dies once the fade reaches zero

		return
	}

	b.kill(inst, DecisionStopped)
}

// StopAll stops everything immediately.
func (b *Bank) StopAll() {
	for _, q := range append([]*Instance(nil), b.queue...) {
		b.kill(q, DecisionStopped)
	}
}

// SetPan sets a fixed pan (-1..1) on a non-positional instance.
func (b *Bank) SetPan(inst *Instance, pan float64) {
	if inst == nil {
		return
	}

	inst.manPan = &pan

	if inst.st == stPlaying {
		b.applyMix(inst)
	}
}

// Advance runs an update pass at the clock's current tick.
func (b *Bank) Advance() { b.process(b.clock.Now()) }

func (b *Bank) distance2(req Request) float64 {
	if !req.HasPos {
		return 0
	}

	dx := clamp(req.X-b.lx, maxClamp)
	dy := clamp(2*(req.Y-b.ly), maxClamp) // y doubled (verified, FUN_004b6110)

	return dx*dx + dy*dy // squared is inferred from the comparison with radius^2
}

func clamp(v, lim float64) float64 { return math.Max(-lim, math.Min(lim, v)) }

func (b *Bank) reject(inst *Instance, d Decision) *Instance {
	inst.st = stDead
	inst.report.Decision = d
	b.emit(inst.report)

	return inst
}

func (b *Bank) emit(r Report) {
	if b.OnReport != nil {
		b.OnReport(r)
	}
}

// less reports whether a is less important than c (victim ordering, derived
// from FUN_004b66f0 and the queue sort): lower priority, then farther, then older.
func less(a, c *Instance) bool {
	if a.priority != c.priority {
		return a.priority < c.priority
	}

	if a.dist2 != c.dist2 {
		return a.dist2 > c.dist2
	}

	if a.start != c.start {
		return a.start < c.start
	}

	return a.seq < c.seq
}

func (b *Bank) kill(inst *Instance, d Decision) {
	if inst.st == stDead {
		return
	}

	b.release(inst)
	inst.st = stDead
	inst.report.Decision = d
	b.emit(inst.report)

	for i, q := range b.queue {
		if q == inst {
			b.queue = append(b.queue[:i], b.queue[i+1:]...)
			break
		}
	}
}

func (b *Bank) release(inst *Instance) {
	if inst.player != nil {
		inst.player.Stop()
		inst.player = nil
	}

	if inst.voice >= 0 && b.voices[inst.voice] == inst {
		b.voices[inst.voice] = nil
	}

	inst.voice = -1
}

// process is one pass of the sound queue update (FUN_004b6970).
func (b *Bank) process(now int64) {
	// Order: priority desc, nearer first, newer first (verified comparator).
	sort.SliceStable(b.queue, func(i, j int) bool {
		a, c := b.queue[i], b.queue[j]
		if a.priority != c.priority {
			return a.priority > c.priority
		}

		if a.dist2 != c.dist2 {
			return a.dist2 < c.dist2
		}

		return a.start > c.start
	})

	solo := false

	for _, q := range append([]*Instance(nil), b.queue...) {
		if q.st == stDead {
			continue
		}

		b.step(q, now)

		if q.st == stPlaying && q.row.Solo {
			solo = true
		}
	}

	if now != b.lastTick { // ducking moves once per tick (shape verified: +/-2 within 70..100)
		b.lastTick = now

		if solo {
			b.duck -= soloDuckStep
			if b.duck < soloDuckMin {
				b.duck = soloDuckMin
			}
		} else {
			b.duck += soloDuckStep
			if b.duck > duckMax {
				b.duck = duckMax
			}
		}
	}

	for _, q := range b.queue {
		if q.st == stPlaying {
			b.applyMix(q)
		}
	}
}

func (b *Bank) audible(q *Instance) bool {
	if !q.req.HasPos {
		return true
	}

	r := FalloffRadius(q.row.Falloff)

	return q.dist2 <= r*r
}

func (b *Bank) step(q *Instance, now int64) {
	row := q.row
	q.dist2 = b.distance2(q.req)
	b.stepFade(q, now)

	switch q.st {
	case stStopping:
		if row.Loop && row.Duration == 0 && !q.stopFlag { // loops go back to waiting (verified)
			q.st = stQueued
		} else {
			b.kill(q, DecisionStopped)
		}

		return
	case stPlaying:
		finished := q.player == nil || !q.player.IsPlaying()
		over := row.Duration != 0 && now-q.start > int64(row.Duration)

		switch {
		case q.stopFlag && !q.fade.active:
			b.kill(q, DecisionStopped)
		case over || (finished && !row.Loop):
			b.kill(q, DecisionFinished)
		case !b.audible(q):
			b.release(q)
			q.st = stStopping
		}

		return
	case stDead:
		return
	case stQueued:
	}

	if now < q.start || q.vol == 0 {
		return
	}

	if !b.audible(q) {
		q.report.Decision = DecisionInaudible
		q.report.Dist2 = q.dist2

		if !row.Loop { // out-of-range one-shots are dropped (verified: no voice and not Loop)
			b.kill(q, DecisionInaudible)
		}

		return
	}

	b.start(q, now)

	if q.st == stQueued && !q.row.Loop {
		b.kill(q, q.report.Decision)
	}
}

func (b *Bank) stepFade(q *Instance, now int64) {
	if !q.fade.active {
		return
	}

	f := q.fade

	switch {
	case now >= f.endTk:
		q.vol = f.to
		q.fade.active = false
	case now <= f.startTk:
		q.vol = f.from
	default:
		q.vol = f.from + int(int64(f.to-f.from)*(now-f.startTk)/(f.endTk-f.startTk))
	}
}

func (b *Bank) start(q *Instance, now int64) {
	orig := q.index

	if !q.req.FixedVariant {
		q.index = b.PickVariant(q.index)
		q.row = b.table.Get(q.index)
		q.report.Picked, q.report.Handle, q.report.File = q.index, q.row.Handle, q.row.FileName
	}

	row := q.row

	// Defer / Stop Inst (shape verified, FUN_004b5fd0): the oldest playing instance of the
	// same group head (same emitter when one is given) is the one affected.
	if row.DeferInst || row.StopInst {
		if old := b.oldestInGroup(q); old != nil && (!row.DeferInst || q.req.Emitter != 0 || !q.req.HasPos) {
			if !row.StopInst {
				b.kill(q, DecisionDeferred) // Defer Inst: the new request yields

				return
			}

			b.kill(old, DecisionStopped) // Stop Inst: the older one is cut, the new one starts
		}
	}

	// Same sound within a tick of a playing copy at >= priority is dropped (verified).
	if o := b.oldestOfIndex(q.index); o != nil && uint64(q.start-o.start) < 2 && q.priority <= o.priority {
		b.kill(q, DecisionSameTick)

		return
	}

	h := b.history[orig]
	b.history[orig] = [2]int{q.index, h[0]}

	slot, victim := b.allocVoice(q)
	if slot < 0 {
		q.report.Decision = DecisionNoVoice

		return
	}

	p, err := b.load(row)
	if err != nil || p == nil {
		b.kill(q, DecisionLoadFailed)

		return
	}

	d := DecisionPlayed

	if victim != nil {
		d = DecisionStolen
		q.report.Victim = victim.row.Handle
		b.release(victim)
		victim.st = stStopping // loops requeue next pass, one-shots die
		victim.report.Decision = DecisionStopped
		b.emit(victim.report)
	}

	q.player, q.voice, q.st = p, slot, stPlaying
	b.voices[slot] = q

	if q.start < now {
		q.start = now
	}

	if row.FadeIn > 0 && !q.req.NoFade && !q.fade.active {
		q.vol = 0
		q.fade = fade{active: true, from: 0, to: 255, startTk: now, endTk: now + int64(row.FadeIn)}
	}

	q.report.Voice, q.report.Decision, q.report.Dist2 = slot, d, q.dist2
	q.lastVol, q.lastPan = -1, 2
	p.Play()
	b.applyMix(q)
	b.emit(q.report)
}

// allocVoice finds a free voice or the least important victim the newcomer may
// take (FUN_004dca60). Equal priority is resolved in favour of the newer
// request; the original compares object addresses there (unverified proxy).
func (b *Bank) allocVoice(q *Instance) (int, *Instance) {
	for i, v := range b.voices {
		if v == nil {
			return i, nil
		}
	}

	vi := 0

	for i, v := range b.voices {
		if less(v, b.voices[vi]) {
			vi = i
		}
	}

	v := b.voices[vi]
	if q.priority > v.priority || (q.priority == v.priority && q.seq > v.seq) {
		return vi, v
	}

	return -1, nil
}

func (b *Bank) oldestInGroup(q *Instance) *Instance {
	head := b.table.Head(q.index)

	var best *Instance

	for _, o := range b.queue {
		if o == q || o.st != stPlaying || b.table.Head(o.index) != head {
			continue
		}

		if q.req.Emitter != 0 && o.req.Emitter != q.req.Emitter {
			continue
		}

		if best == nil || o.start < best.start {
			best = o
		}
	}

	return best
}

func (b *Bank) oldestOfIndex(index int) *Instance {
	var best *Instance

	for _, o := range b.voices {
		if o != nil && o.index == index && (best == nil || o.start < best.start) {
			best = o
		}
	}

	return best
}

// Mix computes the player volume and pan for an instance: vol/255 * row.Volume/255
// * master (music master for MusicVol rows) * solo duck (non-Solo rows) * distance
// gain. The row.Volume factor, the distance gain and the pan model approximate the
// original's 3D positional audio (unverified); the 0.003125 position scale and the
// z of 640 are verified.
func (b *Bank) Mix(q *Instance) (vol, pan float64) {
	row := q.row
	master := b.sfxVol

	if row.MusicVol {
		master = b.musicVol
	}

	vol = float64(q.vol) / 255 * float64(row.Volume) / 255 * master

	if !row.Solo {
		vol *= float64(b.duck) / duckMax
	}

	if q.req.HasPos {
		vol *= DistanceGain(q.dist2, row.Falloff)

		const scale, z = 0.003125, 640.0

		px := clamp(q.req.X-b.lx, maxClamp) * scale
		pan = px / math.Hypot(px, z*scale)
	} else if q.manPan != nil {
		pan = *q.manPan
	}

	return vol, pan
}

// DistanceGain is a linear roll-off to the Falloff radius (unverified).
func DistanceGain(dist2 float64, falloff int) float64 {
	r := FalloffRadius(falloff)
	if dist2 >= r*r {
		return 0
	}

	return 1 - math.Sqrt(dist2)/r
}

func (b *Bank) applyMix(q *Instance) {
	if q.player == nil {
		return
	}

	v, p := b.Mix(q)
	if v != q.lastVol {
		q.player.SetVolume(v)
		q.lastVol = v
	}

	if p != q.lastPan {
		q.player.SetPan(p)
		q.lastPan = p
	}
}
