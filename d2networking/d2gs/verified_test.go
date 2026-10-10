package d2gs

import (
	"bytes"
	"reflect"
	"testing"
)

// Fixtures below were packed with an independent big-integer packer from the
// field widths read in the exe (0x4591b0 for 0x95, 0x459010 for 0x18).
var (
	lifeMana95 = []byte{0x95, 0x34, 0x12, 0x5e, 0x05, 0xfa, 0x80, 0x7d, 0x02, 0x8a, 0xa2, 0x60, 0x1f}
	vitals18   = []byte{0x18, 0x34, 0x12, 0x5e, 0x05, 0xfa, 0x80, 0xbc, 0x61, 0x9f, 0x80, 0xa2, 0x00, 0x0c, 0x04}
)

func TestLifeMana95(t *testing.T) {
	want := LifeMana{Life: 4660, Mana: 2748, Stamina: 1000, X: 5100, Y: 5200, DX: 5, DY: -5}

	got, err := ParseLifeMana(lifeMana95)
	if err != nil || got != want {
		t.Fatalf("parse: %+v %v want %+v", got, err, want)
	}

	if out := want.Marshal(); !bytes.Equal(out, lifeMana95) {
		t.Fatalf("marshal: % x want % x", out, lifeMana95)
	}

	if _, err := ParseLifeMana(lifeMana95[:12]); err == nil {
		t.Fatal("short packet accepted")
	}
}

func TestPlayerVitals18(t *testing.T) {
	// dx byte 0x80 stays +128 and 0x81 becomes -127 (the exe only subtracts 0x100 above 0x80)
	want := PlayerVitals{Life: 4660, Mana: 2748, Stamina: 1000, StatA: 100, StatB: 27, X: 5100, Y: 5200, DX: 128, DY: -127}

	got, err := ParsePlayerVitals(vitals18)
	if err != nil || got != want {
		t.Fatalf("parse: %+v %v want %+v", got, err, want)
	}

	if out := want.Marshal(); !bytes.Equal(out, vitals18) {
		t.Fatalf("marshal: % x want % x", out, vitals18)
	}
}

func TestVitalsRoundTripExtremes(t *testing.T) {
	m := LifeMana{Life: 0x7fff, Mana: 0x7fff, Stamina: 0x7fff, X: 0xffff, Y: 0xffff, DX: -127, DY: 128}
	if g, err := ParseLifeMana(m.Marshal()); err != nil || g != m {
		t.Fatalf("%+v %v", g, err)
	}

	// the 3 spare bits of 0x95 stay zero even when every field is full
	if b := m.Marshal(); b[12]>>5 != 0 {
		t.Fatalf("spare bits set: %#x", b[12])
	}

	v := PlayerVitals{Life: 0x7fff, Mana: 1, Stamina: 0x7fff, StatA: 0x7f, StatB: 0x7f, X: 0xffff, Y: 1, DX: 0, DY: 1}
	if g, err := ParsePlayerVitals(v.Marshal()); err != nil || g != v {
		t.Fatalf("%+v %v", g, err)
	}
}

func TestAssignPlayer59(t *testing.T) {
	raw := make([]byte, 26)
	raw[0], raw[1], raw[2], raw[3], raw[4] = 0x59, 0x78, 0x56, 0x34, 0x12
	raw[5] = 3
	copy(raw[6:], "Maricon")
	raw[22], raw[23], raw[24], raw[25] = 0xec, 0x13, 0x50, 0x14

	want := AssignPlayer{UnitID: 0x12345678, Class: 3, Name: "Maricon", X: 5100, Y: 5200}

	got, err := ParseAssignPlayer(raw)
	if err != nil || got != want {
		t.Fatalf("%+v %v", got, err)
	}

	if !bytes.Equal(want.Marshal(), raw) {
		t.Fatalf("marshal: % x", want.Marshal())
	}
}

func TestVerifiedClientPackets(t *testing.T) {
	tests := []struct {
		name string
		raw  []byte
		msg  Message
	}{
		{"quest flag", []byte{0x58, 0x29, 0x00}, &SetQuestFlag{Index: 41}},
		{"npc interact params", []byte{0x59, 1, 0, 0, 0, 9, 0, 0, 0, 0x10, 0, 0, 0, 0x20, 0, 0, 0}, &NpcInteractParams{1, 9, 16, 32}},
		{"checked move", []byte{0x5f, 0x2c, 0x01, 0xf4, 0x01}, &MoveChecked{X: 300, Y: 500}},
		// 0x3a: u16 = stat | (count-1)<<8
		{"allocate 5 points", []byte{0x3a, 0x03, 0x04}, &AllocateStat{Stat: 3, Extra: 4}},
		{"npc trade", []byte{0x38, 1, 0, 0, 0, 9, 0, 0, 0, 7, 0, 0, 0}, &NpcTrade{1, 9, 7}},
		{"left skill on unit flag0", []byte{0x07, 1, 0, 0, 0, 9, 0, 0, 0}, &UnitOrder{ID: 7, UnitType: 1, UnitID: 9}},
		{"right skill on unit ex", []byte{0x10, 1, 0, 0, 0, 9, 0, 0, 0}, &UnitOrder{ID: 0x10, UnitType: 1, UnitID: 9}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if !bytes.Equal(tc.msg.MarshalPacket(), tc.raw) {
				t.Fatalf("encode % x", tc.msg.MarshalPacket())
			}

			got, err := DecodeClient(tc.raw)
			if err != nil || !reflect.DeepEqual(got, tc.msg) {
				t.Fatalf("decode %+v %v", got, err)
			}

			if err := Validate(ClientToServer, tc.raw); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAllocateStatRules(t *testing.T) {
	if !(AllocateStat{Stat: 0x0f, Extra: 99}).Valid() || (AllocateStat{Stat: 0x10}).Valid() || (AllocateStat{Extra: 100}).Valid() {
		t.Fatal("range checks differ from the handler")
	}

	if (AllocateStat{Extra: 99}).Count() != 100 {
		t.Fatal("count")
	}
}

func TestBitReaderLSBFirst(t *testing.T) {
	r := bitReader{b: []byte{0xb5, 0x01}}
	if v := r.read(4); v != 0x5 {
		t.Fatalf("low nibble first: %#x", v)
	}

	if v := r.read(5); v != 0x1b { // bits 4..8 = 1011 | 1<<4
		t.Fatalf("got %#x", v)
	}

	if v := r.read(32); v != 0 { // past the end reads zeros
		t.Fatalf("got %#x", v)
	}
}

func FuzzVitalsParsers(f *testing.F) {
	f.Add(lifeMana95)
	f.Add(vitals18)
	f.Add(AssignPlayer{UnitID: 1, Name: "n"}.Marshal())
	f.Fuzz(func(t *testing.T, b []byte) {
		if m, err := ParseLifeMana(b); err == nil {
			// ignoring the spare bits and the 0x80 delta quirk, re-encoding must reparse equal
			if g, err := ParseLifeMana(m.Marshal()); err != nil || g != m {
				t.Fatalf("%+v vs %+v (%v)", m, g, err)
			}
		}

		if m, err := ParsePlayerVitals(b); err == nil {
			if g, err := ParsePlayerVitals(m.Marshal()); err != nil || g != m {
				t.Fatalf("%+v vs %+v (%v)", m, g, err)
			}
		}

		if m, err := ParseAssignPlayer(b); err == nil {
			if g, err := ParseAssignPlayer(m.Marshal()); err != nil || g.UnitID != m.UnitID || g.Class != m.Class || g.X != m.X || g.Y != m.Y || len(g.Name) > len(m.Name) {
				t.Fatalf("%+v vs %+v (%v)", m, g, err)
			}
		}
	})
}
