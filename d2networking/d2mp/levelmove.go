package d2mp

// Level-move rate limit. The original remembers the game frames of a hero's
// last five level moves (PLAYER_RecordLevelMoveFrame 0x536db0,
// PLAYER_IsLevelMoveRateLimited 0x536e40). When all five lie within 2251
// frames the move is "limited" and the protection state 0x6c that every move
// grants (125 frames) is replaced by a longer one drawn from a weighted table
// (SERVER_PerformMoveToLevelPosition 0x54aa60, table at 0x6e235c, VERIFIED by
// reading memory: weights 50/25/25 with 1500/3000/4500 frames; the pairing of
// roll ranges to rows is the order of the table, UNVERIFIED).

const (
	levelMoveSlots     = 5
	levelMoveWindow    = 2251 // frames, about 90 s at 25 Hz
	protectionDefault  = 125  // frames of state 0x6c after an ordinary move
	protectionStateID  = 0x6c
	levelMoveRollRange = 100
)

// protectionRow is one weighted entry of the exe's table.
type protectionRow struct{ weight, frames int }

var protectionTable = [...]protectionRow{{50, 1500}, {25, 3000}, {25, 4500}}

// levelMoves is the ring of frame stamps of the last level moves (0 = empty).
type levelMoves struct{ stamps [levelMoveSlots]uint32 }

// record stores a move at frame (>= 1): into an empty slot, else into one
// older than the window, else nothing (the ring is full of recent moves).
func (l *levelMoves) record(frame uint32) {
	for i, f := range l.stamps {
		if f == 0 || frame-f > levelMoveWindow {
			l.stamps[i] = frame

			return
		}
	}
}

// limited reports whether all slots hold moves within the window.
func (l *levelMoves) limited(frame uint32) bool {
	for _, f := range l.stamps {
		if f == 0 || frame-f > levelMoveWindow {
			return false
		}
	}

	return true
}

// ProtectionFrames is the length of state 0x6c for a hero moving at frame:
// 125 normally, one of the table's lengths when the limit is hit. roll is a
// draw in 0..99 and is only consulted when limited.
func (l *levelMoves) protectionFrames(frame uint32, roll func() int) int {
	l.record(frame)

	if !l.limited(frame) {
		return protectionDefault
	}

	r := roll() % levelMoveRollRange
	for _, row := range protectionTable {
		if r < row.weight {
			return row.frames
		}

		r -= row.weight
	}

	return protectionDefault
}

// protectAfterMove records a level move of p and starts state 0x6c.
func (s *Sim) protectAfterMove(p *pstate) {
	tick := s.cfg.TickMs
	frame := s.now/tick + 1
	n := p.moves.protectionFrames(frame, func() int { return s.rng.Intn(levelMoveRollRange) })
	p.protectUntil = s.now + uint32(n)*tick
}
