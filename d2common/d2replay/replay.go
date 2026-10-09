package d2replay

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
)

// Input is one recorded player input, applied before the frame's step.
type Input struct {
	Frame int    `json:"frame"`
	Kind  string `json:"kind"` // e.g. "move", "attack", "cast"
	Actor uint32 `json:"actor"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Arg   int    `json:"arg,omitempty"`
}

// Log is a recorded session: the seed, the frame count and the inputs.
type Log struct {
	Seed   uint32  `json:"seed"`
	Frames int     `json:"frames"`
	Inputs []Input `json:"inputs"`
}

// Recorder collects inputs while a session runs.
type Recorder struct{ log Log }

// NewRecorder starts a log for a session with the given seed.
func NewRecorder(seed uint32) *Recorder { return &Recorder{log: Log{Seed: seed}} }

// Record adds an input at the given frame.
func (r *Recorder) Record(in Input) {
	r.log.Inputs = append(r.log.Inputs, in)

	if in.Frame+1 > r.log.Frames {
		r.log.Frames = in.Frame + 1
	}
}

// Finish fixes the total frame count (at least as many as the last input
// needs) and returns the log.
func (r *Recorder) Finish(frames int) Log {
	if frames > r.log.Frames {
		r.log.Frames = frames
	}

	return r.log
}

// Write encodes the log as JSON.
func (l Log) Write(w io.Writer) error { return json.NewEncoder(w).Encode(l) }

// Read decodes a log written by Write.
func Read(r io.Reader) (Log, error) {
	var l Log

	err := json.NewDecoder(r).Decode(&l)

	return l, err
}

// Sim is the simulation under test. It is created from the seed, advanced one
// frame at a time and must be able to summarise its state in a hash.
type Sim interface {
	// Step applies the frame's inputs (in recorded order) and advances one frame.
	Step(frame int, inputs []Input)
	// Hash writes the full simulation state into h in a canonical order
	// (never in map iteration order).
	Hash(h *Hasher)
}

// Factory builds a fresh simulation from a seed.
type Factory func(seed uint32) Sim

// Run replays the log against a fresh simulation and returns the state hash
// after every frame.
func Run(newSim Factory, l Log) []uint64 {
	sim := newSim(l.Seed)
	byFrame := map[int][]Input{}

	for _, in := range l.Inputs {
		byFrame[in.Frame] = append(byFrame[in.Frame], in)
	}

	trace := make([]uint64, 0, l.Frames)

	for f := 0; f < l.Frames; f++ {
		sim.Step(f, byFrame[f])

		h := NewHasher()
		sim.Hash(h)
		trace = append(trace, h.Sum())
	}

	return trace
}

// Compare returns the first frame at which two traces differ, or -1 when they
// are identical.
func Compare(a, b []uint64) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		if i >= len(a) || i >= len(b) || a[i] != b[i] {
			return i
		}
	}

	return -1
}

// Verify runs the log twice on fresh simulations and reports the first
// diverging frame as an error.
func Verify(newSim Factory, l Log) error {
	if f := Compare(Run(newSim, l), Run(newSim, l)); f >= 0 {
		return fmt.Errorf("replay diverged at frame %d (seed %d)", f, l.Seed)
	}

	return nil
}

// Hasher is a 64-bit FNV-1a state hasher with typed writers, so states hash
// the same on every platform.
type Hasher struct{ h hash.Hash64 }

// NewHasher returns an empty hasher.
func NewHasher() *Hasher { return &Hasher{h: fnv.New64a()} }

// U64 hashes a 64-bit value.
func (h *Hasher) U64(v uint64) {
	var b [8]byte

	binary.LittleEndian.PutUint64(b[:], v)
	_, _ = h.h.Write(b[:])
}

// I hashes an int.
func (h *Hasher) I(v int) { h.U64(uint64(int64(v))) }

// U32 hashes a 32-bit value.
func (h *Hasher) U32(v uint32) { h.U64(uint64(v)) }

// Str hashes a length-prefixed string.
func (h *Hasher) Str(s string) {
	h.I(len(s))
	_, _ = h.h.Write([]byte(s))
}

// Sum returns the hash so far.
func (h *Hasher) Sum() uint64 { return h.h.Sum64() }
