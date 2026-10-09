package d2gs

import (
	"bytes"
	"testing"
)

// Hand-written fixtures laid out from the handler offsets in
// d2-re-notes/verify-roster-packets.md (not produced by the encoders).
var (
	roster5b = append(append([]byte{
		0x5b, 0x2f, 0x00, // id, total length 47
		0x78, 0x56, 0x34, 0x12, // unit id
		0x03,                                                       // class
		'A', 'm', 'a', 'z', 'o', 'n', 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // name[16]
		0x2c, 0x00, // level 44
		0x07, 0x00, // party 7
		0x11, 0x11, // @1c
		0x22, 0x02, // @1e flags 0x222
		0x33, 0x03, // @20
	}, []byte("acct\x00")...), []byte("xyz\x00")...)
	// 0x22 + 5 + 4 = 43; fix the length byte in init
	kills65  = []byte{0x65, 0x01, 0x00, 0x00, 0x80, 0xfe, 0xff}
	update75 = []byte{0x75, 0x78, 0x56, 0x34, 0x12, 0x07, 0x00, 0x2c, 0x00, 0xaa, 0xbb, 0x22, 0x02}
	rel8c    = []byte{0x8c, 0x01, 0, 0, 0, 0x02, 0, 0, 0, 0x05, 0x01}
	party8d  = []byte{0x8d, 0x01, 0, 0, 0, 0xff, 0xff}
	link8e   = []byte{0x8e, 0x01, 0x0a, 0, 0, 0, 0x0b, 0, 0, 0}
)

func init() { roster5b[1] = byte(len(roster5b)) }

func TestRosterEntry5b(t *testing.T) {
	got, err := ParseRosterEntry(roster5b)
	want := RosterEntry{UnitID: 0x12345678, Class: 3, Name: "Amazon", Level: 44, PartyID: 7, Unk1C: 0x1111, Flags: 0x222,
		Unk20: 0x333, Account: "acct", Extra: "xyz"}

	if err != nil || got != want {
		t.Fatalf("%+v %v want %+v", got, err, want)
	}

	if out := want.Marshal(); !bytes.Equal(out, roster5b) {
		t.Fatalf("marshal % x want % x", out, roster5b)
	}

	// the framing rule agrees with the length field
	if n, need, err := serverLength(roster5b); err != nil || need != 0 || n != len(roster5b) {
		t.Fatalf("framing %d %d %v", n, need, err)
	}

	for _, bad := range [][]byte{roster5b[:30], append(append([]byte{}, roster5b...), 0)} {
		if _, err := ParseRosterEntry(bad); err == nil {
			t.Fatalf("accepted % x", bad)
		}
	}

	// missing account terminator
	noNul := append([]byte{}, roster5b[:0x22]...)
	noNul = append(noNul, 'a')
	noNul[1] = byte(len(noNul))

	if _, err := ParseRosterEntry(noNul); err == nil {
		t.Fatal("unterminated account accepted")
	}

	// third string absent reads as empty
	short := append([]byte{}, roster5b[:0x22]...)
	short = append(short, 'a', 0)
	short[1] = byte(len(short))

	if g, err := ParseRosterEntry(short); err != nil || g.Account != "a" || g.Extra != "" {
		t.Fatalf("%+v %v", g, err)
	}
}

func TestRosterFixed(t *testing.T) {
	if g, err := ParseRosterKills(kills65); err != nil || g != (RosterKills{UnitID: 0x80000001, Kills: -2}) {
		t.Fatalf("%+v %v", g, err)
	}

	if !bytes.Equal((RosterKills{UnitID: 0x80000001, Kills: -2}).Marshal(), kills65) {
		t.Fatal("kills marshal")
	}

	u := RosterUpdate{UnitID: 0x12345678, PartyID: 7, Level: 44, Unk9: 0xbbaa, Flags: 0x222}
	if g, err := ParseRosterUpdate(update75); err != nil || g != u || !bytes.Equal(u.Marshal(), update75) {
		t.Fatalf("%+v %v", g, err)
	}

	r := RosterRelation{A: 1, B: 2, Flags: 0x105}
	if g, err := ParseRosterRelation(rel8c); err != nil || g != r || !bytes.Equal(r.Marshal(), rel8c) {
		t.Fatalf("%+v %v", g, err)
	}

	p := RosterParty{UnitID: 1, PartyID: NoParty}
	if g, err := ParseRosterParty(party8d); err != nil || g != p || !bytes.Equal(p.Marshal(), party8d) {
		t.Fatalf("%+v %v", g, err)
	}

	l := RosterLink{Add: true, A: 10, B: 11}
	if g, err := ParseRosterLink(link8e); err != nil || g != l || !bytes.Equal(l.Marshal(), link8e) {
		t.Fatalf("%+v %v", g, err)
	}

	for _, f := range []func([]byte) error{
		func(b []byte) error { _, e := ParseRosterKills(b); return e },
		func(b []byte) error { _, e := ParseRosterUpdate(b); return e },
		func(b []byte) error { _, e := ParseRosterRelation(b); return e },
		func(b []byte) error { _, e := ParseRosterParty(b); return e },
		func(b []byte) error { _, e := ParseRosterLink(b); return e },
		func(b []byte) error { _, e := ParseEventMessage(b); return e },
	} {
		if f([]byte{0}) == nil || f(nil) == nil {
			t.Fatal("accepted garbage")
		}
	}
}

func TestEventMessage5a(t *testing.T) {
	raw := make([]byte, 40)
	raw[0], raw[1], raw[2] = 0x5a, 2, 9
	raw[3], raw[7] = 0x10, 1
	copy(raw[8:], "Sorc")
	copy(raw[24:], []byte{5, 0})

	g, err := ParseEventMessage(raw)
	if err != nil || g.Type != 2 || g.B2 != 9 || g.Param != 0x10 || g.Kind != 1 || g.Name != "Sorc" || g.Name2[0] != 5 {
		t.Fatalf("%+v %v", g, err)
	}

	if !bytes.Equal(g.Marshal(), raw) {
		t.Fatalf("marshal % x", g.Marshal())
	}
}

func TestRosterSizesMatchTable(t *testing.T) {
	for id, want := range map[byte]int{S2CEventMessage: 40, S2CRosterKills: 7, S2CRosterUpdate: 13, S2CRosterRelate: 11,
		S2CRosterParty: 7, S2CRosterLink: 10} {
		if n, k := ExpectedSize(id, ServerToClient); k != SizeFixed || n != want {
			t.Errorf("%#x: %d %v", id, n, k)
		}
	}

	if _, k := ExpectedSize(S2CRosterEntry, ServerToClient); k != SizeVariable {
		t.Error("0x5b should be variable")
	}
}

func FuzzRosterParsers(f *testing.F) {
	for _, s := range [][]byte{roster5b, kills65, update75, rel8c, party8d, link8e, EventMessage{Type: 2, Name: "n"}.Marshal()} {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, b []byte) {
		if m, err := ParseRosterEntry(b); err == nil {
			if g, err := ParseRosterEntry(m.Marshal()); err != nil || g != m {
				t.Fatalf("%+v vs %+v (%v)", m, g, err)
			}
		}

		if m, err := ParseRosterKills(b); err == nil && !bytes.Equal(m.Marshal(), b) {
			t.Fatal("kills")
		}

		if m, err := ParseRosterUpdate(b); err == nil && !bytes.Equal(m.Marshal(), b) {
			t.Fatal("update")
		}

		if m, err := ParseRosterRelation(b); err == nil && !bytes.Equal(m.Marshal(), b) {
			t.Fatal("relation")
		}

		if m, err := ParseRosterParty(b); err == nil && !bytes.Equal(m.Marshal(), b) {
			t.Fatal("party")
		}

		if m, err := ParseRosterLink(b); err == nil {
			if g, err := ParseRosterLink(m.Marshal()); err != nil || g != m {
				t.Fatal("link")
			}
		}

		if m, err := ParseEventMessage(b); err == nil {
			if g, err := ParseEventMessage(m.Marshal()); err != nil || g != m {
				t.Fatal("event")
			}
		}
	})
}
