// Package d2unitmsg models the deferred per-unit server message queue of
// Game.exe (UNIT\SUnitMsg.cpp) and the hit-sequence list of UNIT\SUnitDmg.cpp.
//
// A server routine that wants to tell the clients about a unit does not send at
// once: it appends a small message to a first-in-first-out list on the unit
// (types 0x99 item-proc cast, 0x9e merc stat, 0xab, 0xa5, 0x23 ...) and puts
// the unit on its room's dirty list; the room flush later drains the list in
// order. VERIFIED by decompile: the append order (head and tail pointers), the
// 0xa5 message (unit type or 6 for a null unit, unit id, a u16; queueing it
// first clears unit state 0x12) and the hit-sequence purge
// (0x57a900). UNVERIFIED: the exact flush point and the 0x23 layout.
//
// The package is pure and NOT wired: the Go simulation (d2networking/d2mp,
// d2core/d2monsters) emits its events immediately. It gives that code the
// ordering and cleanup rules to adopt.
package d2unitmsg

// Message types.
const (
	TypeCastSkillAtUnit = 0x99
	TypeMercStat        = 0x9e
	TypeUnitByte        = 0xab
	TypeSUnit           = 0xa5
	TypeA3              = 0xa3
	TypeA4              = 0xa4
	Type23              = 0x23

	// StateClearedBySUnit is the unit state removed when a 0xa5 message is
	// queued (whirlwind start/end, charge and leap queue it).
	StateClearedBySUnit = 0x12

	// NoUnit is the unit type written for a null unit.
	NoUnit = 6
	// NoID is the unit id written for a null unit.
	NoID = 0xffffffff
)

// Message is one queued message. Which fields are used depends on the type.
type Message struct {
	Type     byte
	UnitType byte
	UnitID   uint32
	Value    uint32 // skill id, stat value, or the u16 of 0xa5
	Extra    uint32 // target id, stat index ...
}

// Unit is what the queue needs of a unit.
type Unit struct {
	Type byte
	ID   uint32
	// States holds the unit's active states; the queue clears 0x12 here.
	States map[int]bool

	queue []Message
}

// Dirty is the room dirty list: units that have pending messages, in the order
// they were first queued.
type Dirty struct {
	units []*Unit
	in    map[*Unit]bool
}

// NewDirty returns an empty dirty list.
func NewDirty() *Dirty { return &Dirty{in: map[*Unit]bool{}} }

func (d *Dirty) add(u *Unit) {
	if !d.in[u] {
		d.in[u] = true
		d.units = append(d.units, u)
	}
}

// ident returns the (type, id) a message carries for a unit (a nil unit is
// type 6, id 0xffffffff).
func ident(u *Unit) (byte, uint32) {
	if u == nil {
		return NoUnit, NoID
	}

	return u.Type, u.ID
}

// Queue appends a message to the unit (FIFO) and marks the unit dirty.
func (d *Dirty) Queue(u *Unit, m Message) {
	u.queue = append(u.queue, m)
	d.add(u)
}

// QueueSUnit queues a 0xa5 message for the unit. As in the exe it first
// removes state 0x12 from the unit.
func (d *Dirty) QueueSUnit(u *Unit, v uint16) {
	delete(u.States, StateClearedBySUnit)

	t, id := ident(u)
	d.Queue(u, Message{Type: TypeSUnit, UnitType: t, UnitID: id, Value: uint32(v)})
}

// QueueCastSkillAtUnit queues a 0x99 message (an item proc casting a skill).
func (d *Dirty) QueueCastSkillAtUnit(u *Unit, skill uint16, target uint32) {
	t, id := ident(u)
	d.Queue(u, Message{Type: TypeCastSkillAtUnit, UnitType: t, UnitID: id, Value: uint32(skill), Extra: target})
}

// QueueMercStat queues a 0x9e message. The exe is fatal for a stat index above
// 0xfe; ok is false then and nothing is queued.
func (d *Dirty) QueueMercStat(u *Unit, stat int, value uint32) (ok bool) {
	if stat < 0 || stat > 0xfe {
		return false
	}

	t, id := ident(u)
	d.Queue(u, Message{Type: TypeMercStat, UnitType: t, UnitID: id, Value: value, Extra: uint32(stat)})

	return true
}

// Pending returns the number of messages waiting on the unit.
func (u *Unit) Pending() int { return len(u.queue) }

// Flush drains every dirty unit, units in the order they became dirty and each
// unit's messages in queue order, calling send for each. The dirty list is
// emptied.
func (d *Dirty) Flush(send func(u *Unit, m Message)) {
	units := d.units
	d.units, d.in = nil, map[*Unit]bool{}

	for _, u := range units {
		q := u.queue
		u.queue = nil

		for _, m := range q {
			send(u, m)
		}
	}
}

// Hit is one entry of a defender's hit-sequence list: which attacker it came
// from, and the sequence step still to apply.
type Hit struct {
	AttackerType byte
	AttackerID   uint32
	Step         int
}

// HitList is the hit-sequence list of a defender (Unit +0xac).
type HitList struct{ hits []Hit }

// Add appends an entry.
func (l *HitList) Add(h Hit) { l.hits = append(l.hits, h) }

// Len returns the number of entries.
func (l *HitList) Len() int { return len(l.hits) }

// Entries returns a copy of the entries in order.
func (l *HitList) Entries() []Hit { return append([]Hit(nil), l.hits...) }

// PurgeAttacker removes every entry that came from the given attacker, as when
// the attacker is freed (VERIFIED 0x57a900: entries match on type and id; the
// order of the rest is kept). It returns the number removed.
func (l *HitList) PurgeAttacker(attackerType byte, attackerID uint32) int {
	kept := l.hits[:0]
	removed := 0

	for _, h := range l.hits {
		if h.AttackerType == attackerType && h.AttackerID == attackerID {
			removed++

			continue
		}

		kept = append(kept, h)
	}

	l.hits = kept

	return removed
}
