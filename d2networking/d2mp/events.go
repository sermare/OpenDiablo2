package d2mp

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// EvType is the type of an Event. The numbering is OUR extension (nothing like
// it is known to exist in the original protocol, which sends many separate
// fixed-size packets instead); it is carried in the realm tunnel.
type EvType uint8

// Event types.
const (
	EvTick   EvType = iota + 1 // A = server ms; first event of every batch
	EvSpawn                    // Unit (+ Seg) enters the viewer's level
	EvSeg                      // ID starts the segment Seg
	EvAttack                   // ID swings/casts at Other with skill A
	EvHit                      // ID was hit by Other for A, B = hp after
	EvDeath                    // ID died, Other = killer
	EvRemove                   // ID leaves the level (picked up, expired, left)
	EvObject                   // ID is now in state A
	EvLevel                    // viewer's own unit ID is now in Level: drop everything else
	EvVitals                   // ID has hp A of B (respawn, heal)
	EvXP                       // ID gained A experience (B = game total)
	EvParty                    // ID is in party A (0 = none)
	EvTrade                    // trade window state for viewer ID (see Event.Trade)
	EvInv                      // viewer's inventory: Items, gold A
	EvMsg                      // system message Text for the viewer
)

var evNames = map[EvType]string{
	EvTick: "tick", EvSpawn: "spawn", EvSeg: "seg", EvAttack: "attack", EvHit: "hit", EvDeath: "death",
	EvRemove: "remove", EvObject: "object", EvLevel: "level", EvVitals: "vitals", EvXP: "xp", EvParty: "party",
	EvTrade: "trade", EvInv: "inv", EvMsg: "msg",
}

func (t EvType) String() string {
	if n, ok := evNames[t]; ok {
		return n
	}

	return fmt.Sprintf("ev%d", uint8(t))
}

// Trade states in Event.Trade.
const (
	TradeRequested uint8 = iota
	TradeOpen
	TradeDone
	TradeCancelled
)

// TradeView is the trade window as one viewer sees it.
type TradeView struct {
	State                 uint8
	Partner               uint32
	MyItems, TheirItems   []string
	MyGold, TheirGold     uint32
	MyAccept, TheirAccept bool
	Incoming              bool // state requested: the viewer is asked
	Reason                string
}

// Event is one world change, server to client. Fields are used per type as
// documented on the constants.
type Event struct {
	Type    EvType
	ID      uint32
	Other   uint32
	A, B    int32
	Level   uint16
	Unit    Unit // EvSpawn (Segs holds the first segment)
	Seg     Seg  // EvSeg
	Text    string
	Items   []string
	Trade   TradeView
	Visible bool // server side only: not encoded
}

// ErrBadEvent is returned for truncated or unknown events.
var ErrBadEvent = errors.New("d2mp: malformed event")

type ew struct{ b []byte }

func (w *ew) u8(v uint8)   { w.b = append(w.b, v) }
func (w *ew) u16(v uint16) { w.b = binary.LittleEndian.AppendUint16(w.b, v) }
func (w *ew) u32(v uint32) { w.b = binary.LittleEndian.AppendUint32(w.b, v) }
func (w *ew) i32(v int32)  { w.u32(uint32(v)) }
func (w *ew) bool(v bool) {
	if v {
		w.u8(1)
	} else {
		w.u8(0)
	}
}

func (w *ew) str(s string) {
	if len(s) > 255 {
		s = s[:255]
	}

	w.u8(uint8(len(s)))
	w.b = append(w.b, s...)
}

func (w *ew) strs(l []string) {
	if len(l) > 255 {
		l = l[:255]
	}

	w.u8(uint8(len(l)))

	for _, s := range l {
		w.str(s)
	}
}

func (w *ew) seg(s Seg) {
	w.u16(Sub(s.X0))
	w.u16(Sub(s.Y0))
	w.u16(Sub(s.X1))
	w.u16(Sub(s.Y1))
	w.u16(uint16(s.Speed*100 + 0.5))
	w.u32(s.T0)
}

type er struct {
	b   []byte
	err error
}

func (r *er) take(n int) []byte {
	if r.err != nil || n > len(r.b) {
		r.err = ErrBadEvent

		return make([]byte, 4)
	}

	p := r.b[:n]
	r.b = r.b[n:]

	return p
}

func (r *er) u8() uint8   { return r.take(1)[0] }
func (r *er) u16() uint16 { return binary.LittleEndian.Uint16(r.take(2)) }
func (r *er) u32() uint32 { return binary.LittleEndian.Uint32(r.take(4)) }
func (r *er) i32() int32  { return int32(r.u32()) }
func (r *er) bool() bool  { return r.u8() != 0 }
func (r *er) str() string { return string(r.take(int(r.u8()))) }

func (r *er) strs() []string {
	n := int(r.u8())
	out := make([]string, 0, n)

	for i := 0; i < n && r.err == nil; i++ {
		out = append(out, r.str())
	}

	return out
}

func (r *er) seg() Seg {
	var s Seg
	s.X0, s.Y0, s.X1, s.Y1 = FromSub(r.u16()), FromSub(r.u16()), FromSub(r.u16()), FromSub(r.u16())
	s.Speed = float64(r.u16()) / 100
	s.T0 = r.u32()

	return s
}

func (w *ew) unit(u Unit) {
	w.u32(u.ID)
	w.u8(uint8(u.Kind))
	w.u16(u.Type)
	w.u16(u.Level)
	w.str(u.Name)
	w.u32(u.Owner)
	w.i32(u.HP)
	w.i32(u.MaxHP)
	w.bool(u.Dead)
	w.u8(u.State)
	w.u16(u.Dest)
	w.u16(u.Party)
	w.u16(u.Mon)

	s := Seg{}
	if len(u.Segs) > 0 {
		s = u.Segs[len(u.Segs)-1]
	}

	w.seg(s)
}

func (r *er) unit() Unit {
	var u Unit
	u.ID, u.Kind, u.Type, u.Level = r.u32(), Kind(r.u8()), r.u16(), r.u16()
	u.Name, u.Owner, u.HP, u.MaxHP = r.str(), r.u32(), r.i32(), r.i32()
	u.Dead, u.State, u.Dest, u.Party, u.Mon = r.bool(), r.u8(), r.u16(), r.u16(), r.u16()
	u.Segs = []Seg{r.seg()}

	return u
}

func (w *ew) trade(t TradeView) {
	w.u8(t.State)
	w.u32(t.Partner)
	w.strs(t.MyItems)
	w.strs(t.TheirItems)
	w.u32(t.MyGold)
	w.u32(t.TheirGold)
	w.bool(t.MyAccept)
	w.bool(t.TheirAccept)
	w.bool(t.Incoming)
	w.str(t.Reason)
}

func (r *er) trade() TradeView {
	var t TradeView
	t.State, t.Partner = r.u8(), r.u32()
	t.MyItems, t.TheirItems = r.strs(), r.strs()
	t.MyGold, t.TheirGold = r.u32(), r.u32()
	t.MyAccept, t.TheirAccept, t.Incoming = r.bool(), r.bool(), r.bool()
	t.Reason = r.str()

	return t
}

// AppendEvent appends the encoding of e to b.
func AppendEvent(b []byte, e Event) []byte {
	w := &ew{b: b}
	w.u8(uint8(e.Type))

	switch e.Type {
	case EvTick:
		w.i32(e.A)
	case EvSpawn:
		w.unit(e.Unit)
	case EvSeg:
		w.u32(e.ID)
		w.seg(e.Seg)
	case EvAttack, EvHit:
		w.u32(e.ID)
		w.u32(e.Other)
		w.i32(e.A)
		w.i32(e.B)
	case EvDeath:
		w.u32(e.ID)
		w.u32(e.Other)
	case EvRemove:
		w.u32(e.ID)
	case EvObject, EvParty:
		w.u32(e.ID)
		w.i32(e.A)
	case EvLevel:
		w.u32(e.ID)
		w.u16(e.Level)
	case EvVitals, EvXP:
		w.u32(e.ID)
		w.i32(e.A)
		w.i32(e.B)
	case EvTrade:
		w.u32(e.ID)
		w.trade(e.Trade)
	case EvInv:
		w.u32(e.ID)
		w.i32(e.A)
		w.strs(e.Items)
	case EvMsg:
		w.u32(e.ID)
		w.str(e.Text)
	}

	return w.b
}

// EncodeEvents encodes a batch: u16 count + events.
func EncodeEvents(evs []Event) []byte {
	b := binary.LittleEndian.AppendUint16(nil, uint16(len(evs)))

	for _, e := range evs {
		b = AppendEvent(b, e)
	}

	return b
}

// DecodeEvents decodes a batch made by EncodeEvents.
func DecodeEvents(b []byte) ([]Event, error) {
	r := &er{b: b}
	n := int(r.u16())
	out := make([]Event, 0, n)

	for i := 0; i < n && r.err == nil; i++ {
		e := Event{Type: EvType(r.u8())}

		switch e.Type {
		case EvTick:
			e.A = r.i32()
		case EvSpawn:
			e.Unit = r.unit()
			e.ID = e.Unit.ID
		case EvSeg:
			e.ID = r.u32()
			e.Seg = r.seg()
		case EvAttack, EvHit:
			e.ID, e.Other, e.A, e.B = r.u32(), r.u32(), r.i32(), r.i32()
		case EvDeath:
			e.ID, e.Other = r.u32(), r.u32()
		case EvRemove:
			e.ID = r.u32()
		case EvObject, EvParty:
			e.ID, e.A = r.u32(), r.i32()
		case EvLevel:
			e.ID, e.Level = r.u32(), r.u16()
		case EvVitals, EvXP:
			e.ID, e.A, e.B = r.u32(), r.i32(), r.i32()
		case EvTrade:
			e.ID = r.u32()
			e.Trade = r.trade()
		case EvInv:
			e.ID, e.A = r.u32(), r.i32()
			e.Items = r.strs()
		case EvMsg:
			e.ID = r.u32()
			e.Text = r.str()
		default:
			return nil, fmt.Errorf("%w: type %d", ErrBadEvent, e.Type)
		}

		out = append(out, e)
	}

	if r.err != nil || len(r.b) != 0 {
		return nil, ErrBadEvent
	}

	return out, nil
}

// Command types sent in the tunnel (client to server). Walk, run, cast and
// skill selection use the d2gs packets instead (see package d2realm).
type CmdType uint8

// Command types.
const (
	CmdRespawn      CmdType = iota + 1 // dead hero returns to town
	CmdUseWaypoint                     // A = destination level (the hero stands at an activated waypoint)
	CmdPartyInvite                     // Target = player unit id
	CmdPartyAccept                     // accept the pending invitation
	CmdPartyLeave                      // leave the party
	CmdTradeRequest                    // Target = player unit id
	CmdTradeRespond                    // A = 1 accept the request, 0 decline
	CmdTradeOffer                      // Items + A = gold
	CmdTradeAccept                     // accept the offers as shown
	CmdTradeCancel                     // close the window
	CmdPortal                          // cast a town portal now (also: skill 220 on a location)
)

// Command is one client request that has no d2gs packet.
type Command struct {
	Type   CmdType
	Target uint32
	A      int32
	Items  []string
}

// EncodeCommand serialises a command.
func EncodeCommand(c Command) []byte {
	w := &ew{}
	w.u8(uint8(c.Type))
	w.u32(c.Target)
	w.i32(c.A)
	w.strs(c.Items)

	return w.b
}

// DecodeCommand parses a command.
func DecodeCommand(b []byte) (Command, error) {
	r := &er{b: b}
	c := Command{Type: CmdType(r.u8()), Target: r.u32(), A: r.i32(), Items: r.strs()}

	if r.err != nil || len(r.b) != 0 {
		return Command{}, ErrBadEvent
	}

	return c, nil
}
