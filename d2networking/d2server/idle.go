package d2server

import (
	"sync"
	"time"
)

// Idle tracking of the clients, after Game.exe's CCMD_ProcessClientGameMessage
// (0x53d120) and GAME_KickTimedOutClients (0x52adc0), both VERIFIED by
// decompile: every in-game message stamps the sender's last-message time
// (ClientData +0x3d8), and a periodic sweep kicks clients that fell silent.
//
// The stamp is wired into OnPacketReceived. The sweep (IdleExpired) is exposed
// and tested but NOT wired: the exe runs it only when two global switches are
// set (their meaning is unknown), and the Go server has no kick path that also
// saves the hero first.

// Idle limits of the exe.
const (
	// IdleKickAfter kicks a silent client outright.
	IdleKickAfter = 45 * time.Second
	// IdleSlowKickAfter is the shorter limit of a client flagged as slow
	// (flag test CHARINFO_TestFlagBits, meaning unverified) with a backlog.
	IdleSlowKickAfter = 10 * time.Second
	// IdleBacklogLimit is the backlog (ClientData +0x54, meaning unverified)
	// above which the short limit applies.
	IdleBacklogLimit = 10

	// MaxSaveChunk is the exclusive upper bound of a save-upload chunk (lobby
	// opcode 0x6c: a chunk length of 0x2000 or more is a fatal error).
	MaxSaveChunk = 0x2000
)

// idleTracker holds the last-message time per client; the clock is injectable.
type idleTracker struct {
	mu   sync.Mutex
	now  func() time.Time
	last map[string]time.Time
}

func newIdleTracker(now func() time.Time) *idleTracker {
	if now == nil {
		now = time.Now
	}

	return &idleTracker{now: now, last: map[string]time.Time{}}
}

// Stamp records a message of the client.
func (t *idleTracker) Stamp(id string) {
	t.mu.Lock()
	t.last[id] = t.now()
	t.mu.Unlock()
}

// Forget drops a client that left.
func (t *idleTracker) Forget(id string) {
	t.mu.Lock()
	delete(t.last, id)
	t.mu.Unlock()
}

// Last returns the last message time of a client.
func (t *idleTracker) Last(id string) (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	v, ok := t.last[id]

	return v, ok
}

// IdleExpired is the exe's kick test for one client: the last message must lie
// in the past, and then the client goes when it was silent for more than
// IdleKickAfter, or for more than IdleSlowKickAfter while flagged slow with a
// backlog above IdleBacklogLimit, or when it is marked (ClientData +0x58,
// meaning unverified).
func IdleExpired(now, last time.Time, slow bool, backlog int, marked bool) bool {
	if !last.Before(now) {
		return false
	}

	idle := now.Sub(last)

	return (slow && idle > IdleSlowKickAfter && backlog > IdleBacklogLimit) || idle > IdleKickAfter || marked
}

// SaveChunkOK reports whether a save-upload chunk length is acceptable (the exe
// treats 0x2000 and above as fatal). The Go server sends its saves whole or in
// 4096-byte chunks (chunkSize), well below the cap.
func SaveChunkOK(n int) bool { return n >= 0 && n < MaxSaveChunk }
