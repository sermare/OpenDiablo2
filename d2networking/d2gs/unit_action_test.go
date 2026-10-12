package d2gs

import (
	"bytes"
	"reflect"
	"testing"
)

// TestUnitFamilyServerRoundTrip checks, for every typed message of the unit-action,
// attribute, skill and world families, that Marshal has the exe's size and
// carries the pinned bytes, and that DecodeServer returns the same value.
func TestUnitFamilyServerRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  ServerMessage
		hex  []byte // expected wire bytes
	}{
		{"0c", UnitActionBasic{Unit: UnitRef{1, 0x04030201}, Mode: 5, A: 6, B: 7},
			[]byte{0x0c, 1, 1, 2, 3, 4, 5, 6, 7}},
		{"0d", UnitStop{Unit: UnitRef{0, 0x11223344}, Mode: 2, X: 0x0a0b, Y: 0x0c0d, B11: 1, B12: 99},
			[]byte{0x0d, 0, 0x44, 0x33, 0x22, 0x11, 2, 0x0b, 0x0a, 0x0d, 0x0c, 1, 99}},
		{"0e", ObjectState{Unit: UnitRef{2, 7}, Mode: 3, State: 4, Param: 0xa1b2c3d4},
			[]byte{0x0e, 2, 7, 0, 0, 0, 3, 4, 0xd4, 0xc3, 0xb2, 0xa1}},
		{"10", MoveToEntity{Unit: UnitRef{0, 9}, Mode: 1, TargetType: 1, TargetID: 0x01020304, X: 0x1112, Y: 0x2122},
			[]byte{0x10, 0, 9, 0, 0, 0, 1, 1, 4, 3, 2, 1, 0x12, 0x11, 0x22, 0x21}},
		{"4c", UnitSkillOnTarget{Unit: UnitRef{0, 5}, Skill: 0x0a0b, Level: 3, TargetType: 1, TargetID: 0x0f0e0d0c, Extra: 0x3132},
			[]byte{0x4c, 0, 5, 0, 0, 0, 0x0b, 0x0a, 3, 1, 0x0c, 0x0d, 0x0e, 0x0f, 0x32, 0x31}},
		{"99", UnitSkillOnTarget{Local: true, Unit: UnitRef{0, 5}, Skill: 1, TargetID: 2},
			[]byte{0x99, 0, 5, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 0, 0}},
		{"67", NpcMoveToLocation{ID: 0x01020304, Mode: 2, X: 0x0a0b, Y: 0x0c0d, B10: 1, B11: 2, B12: 3, W13: -2, B15: 9},
			[]byte{0x67, 4, 3, 2, 1, 2, 0x0b, 0x0a, 0x0d, 0x0c, 1, 2, 3, 0xfe, 0xff, 9}},
		{"68", NpcMoveToTarget{ID: 1, Mode: 2, X: 3, Y: 4, TargetType: 1, TargetID: 0x0a0b0c0d, B15: 5, B16: 6, B17: 7, W18: -1, B20: 8},
			[]byte{0x68, 1, 0, 0, 0, 2, 3, 0, 4, 0, 1, 0x0d, 0x0c, 0x0b, 0x0a, 5, 6, 7, 0xff, 0xff, 8}},
		{"69", NpcStateAtLocation{ID: 1, Mode: 2, X: 3, Y: 4, B10: 5, B11: 6},
			[]byte{0x69, 1, 0, 0, 0, 2, 3, 0, 4, 0, 5, 6}},
		{"6a", NpcStateOnTarget{ID: 1, Mode: 2, TargetType: 1, TargetID: 0x0a0b0c0d, B11: 6},
			[]byte{0x6a, 1, 0, 0, 0, 2, 1, 0x0d, 0x0c, 0x0b, 0x0a, 6}},
		{"6b", NpcAttackLocation{NpcStateAtLocation{ID: 1, Mode: 2, X: 3, Y: 4, B10: 5, B11: 6}, 0x0708, 0x090a},
			[]byte{0x6b, 1, 0, 0, 0, 2, 3, 0, 4, 0, 5, 6, 8, 7, 0x0a, 9}},
		{"6c", NpcAttackTarget{NpcStateOnTarget{ID: 1, Mode: 2, TargetType: 1, TargetID: 0x0a0b0c0d, B11: 6}, 0x0708, 0x090a},
			[]byte{0x6c, 1, 0, 0, 0, 2, 1, 0x0d, 0x0c, 0x0b, 0x0a, 6, 8, 7, 0x0a, 9}},
		{"6d", NpcStop{ID: 1, X: 0x0203, Y: 0x0405, B9: 7},
			[]byte{0x6d, 1, 0, 0, 0, 3, 2, 5, 4, 7}},
		{"19", AttrUpdate{ID: 0x19, Value: 200}, []byte{0x19, 200}},
		{"1a", AttrUpdate{ID: 0x1a, Value: 9}, []byte{0x1a, 9}},
		{"1b", AttrUpdate{ID: 0x1b, Value: 0x1234}, []byte{0x1b, 0x34, 0x12}},
		{"1c", AttrUpdate{ID: 0x1c, Value: 0x01020304}, []byte{0x1c, 4, 3, 2, 1}},
		{"1d", AttrUpdate{ID: 0x1d, Stat: 12, Value: 94}, []byte{0x1d, 12, 94}},
		{"1e", AttrUpdate{ID: 0x1e, Stat: 7, Value: 0x1234}, []byte{0x1e, 7, 0x34, 0x12}},
		{"1f", AttrUpdate{ID: 0x1f, Stat: 7, Value: 0x01020304}, []byte{0x1f, 7, 4, 3, 2, 1}},
		{"20", SetPlayerStat{UnitID: 0x01020304, Stat: 0x0c, Value: 0xa1b2c3d4},
			[]byte{0x20, 4, 3, 2, 1, 0x0c, 0xd4, 0xc3, 0xb2, 0xa1}},
		{"21", UpdateSkillLevel{UnitType: 0, Flag: 1, UnitID: 0x01020304, Skill: 0x0a0b, Level: 20},
			[]byte{0x21, 0, 1, 4, 3, 2, 1, 0x0b, 0x0a, 20, 0, 0}},
		{"23", SetActiveSkill{UnitType: 0, UnitID: 0x01020304, Hand: 1, Skill: 0x0a0b, ItemID: 0xffffffff},
			[]byte{0x23, 0, 4, 3, 2, 1, 1, 0x0b, 0x0a, 0xff, 0xff, 0xff, 0xff}},
		{"7b", SkillSlot{Slot: 3, Skill: 0x0123, Flag: true, Param: 0x01020304},
			[]byte{0x7b, 3, 0x23, 0x81, 4, 3, 2, 1}},
		{"63", WaypointMenu{ObjectID: 0x01020304, Flags: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
			[]byte{0x63, 4, 3, 2, 1, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}},
		{"73", MissileCreate{Class: 0x0a0b, X: 0x11, Y: 0x12, TargetX: 0x13, TargetY: 0x14, W23: 0x0203, OwnerType: 1, OwnerID: 0x0a0b0c0d, B30: 7, Stat148: 9},
			[]byte{0x73, 0, 0, 0, 0, 0x0b, 0x0a, 0x11, 0, 0, 0, 0x12, 0, 0, 0, 0x13, 0, 0, 0, 0x14, 0, 0, 0, 3, 2, 1,
				0x0d, 0x0c, 0x0b, 0x0a, 7, 9}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.msg.Marshal()
			if !bytes.Equal(got, tc.hex) {
				t.Fatalf("marshal % x want % x", got, tc.hex)
			}

			// the size must be what the exe's table says
			n, kind := ExpectedSize(got[0], ServerToClient)
			if kind != SizeFixed || n != len(got) {
				t.Fatalf("id %#x: table says %d kind %d, packet is %d bytes", got[0], n, kind, len(got))
			}

			dec, err := DecodeServer(tc.hex)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(dec, tc.msg) {
				t.Fatalf("decode %+v want %+v", dec, tc.msg)
			}

			if !bytes.Equal(dec.Marshal(), tc.hex) {
				t.Fatal("re-encode differs")
			}

			// every cut and every growth is refused
			for cut := 0; cut < len(tc.hex); cut++ {
				if _, err := DecodeServer(tc.hex[:cut]); err == nil {
					t.Fatalf("cut %d accepted", cut)
				}
			}

			if _, err := DecodeServer(append(append([]byte{}, tc.hex...), 0)); err == nil {
				t.Fatal("oversize accepted")
			}
		})
	}
}

func TestSetStatWidth(t *testing.T) {
	for _, tc := range []struct {
		v  uint32
		id byte
	}{{0, 0x1d}, {0xfe, 0x1d}, {0xff, 0x1e}, {0xfffe, 0x1e}, {0xffff, 0x1f}, {1 << 31, 0x1f}} {
		m := SetStat(5, tc.v)
		if m.ID != tc.id {
			t.Errorf("value %#x picks %#x want %#x", tc.v, m.ID, tc.id)
		}

		back, err := ParseAttrUpdate(m.Marshal())
		if err != nil || back != m || back.StatID() != 5 || back.Add() {
			t.Errorf("value %#x: %+v %v", tc.v, back, err)
		}
	}

	for id, want := range map[byte]uint8{0x19: 14, 0x1a: 13, 0x1b: 13, 0x1c: 13} {
		if got := (AttrUpdate{ID: id}).StatID(); got != want {
			t.Errorf("id %#x stat %d want %d", id, got, want)
		}
	}

	if !(AttrUpdate{ID: 0x1b}).Add() || (AttrUpdate{ID: 0x1c}).Add() {
		t.Error("0x1b adds, 0x1c sets")
	}

	if (AttrUpdate{ID: 0x30}).Marshal() != nil {
		t.Error("bad id marshals")
	}
}

func TestDecodeServerRejectsOthers(t *testing.T) {
	if _, err := DecodeServer([]byte{0x00}); err == nil {
		t.Error("untyped id accepted")
	}

	if _, err := DecodeServer(nil); err == nil {
		t.Error("empty accepted")
	}

	if _, err := ParseNpcAttackLocation(NpcStateAtLocation{}.Marshal()); err == nil {
		t.Error("0x69 parsed as 0x6b")
	}
}

// TestMissileHandlerOffsets pins the offsets that the handler 0x459e10 and the
// sender 0x59daa0 share.
func TestMissileHandlerOffsets(t *testing.T) {
	m := MissileCreate{Class: 1, OwnerType: 3, OwnerID: 0xaabbccdd, B30: 0x5a, Stat148: 0x5b}.Marshal()
	if m[5] != 1 || m[25] != 3 || m[26] != 0xdd || m[29] != 0xaa || m[30] != 0x5a || m[31] != 0x5b {
		t.Errorf("% x", m)
	}
}

// TestClientVerifiedSizes pins the client -> server lengths read from the
// handlers in this pass.
func TestClientVerifiedSizes(t *testing.T) {
	want := map[byte]int{
		0x0b: 1, 0x3d: 5, 0x3e: 5, 0x3f: 3, 0x40: 1, 0x41: 1, 0x44: 17, 0x46: 13, 0x47: 13, 0x48: 1,
		0x49: 9, 0x4b: 9, 0x4c: 5, 0x4d: 3, 0x4f: 7, 0x50: 9, 0x53: 1, 0x54: 1, 0x60: 1, 0x61: 3, 0x62: 5, 0x63: 5,
	}

	for id, n := range want {
		got, kind := ExpectedSize(id, ClientToServer)
		if kind != SizeFixed || got != n {
			t.Errorf("C2S %#x: %d kind %d want %d", id, got, kind, n)
		}

		full := make([]byte, n)
		full[0] = id

		msg, err := DecodeClient(full)
		if err != nil {
			t.Fatalf("%#x: %v", id, err)
		}

		if !bytes.Equal(msg.MarshalPacket(), full) {
			t.Errorf("%#x: round trip % x", id, msg.MarshalPacket())
		}
	}
}

func TestVerifiedClientRoundTrip(t *testing.T) {
	for _, m := range []decodable{
		&ObjectMode{ObjectID: 0x01020304}, &QuestItemUse{ItemID: 5}, &SetUnitMode{Mode: 0x1b},
		&ObjectItemUse{ObjectID: 1, ItemID: 2, Action: 3}, &PetCommandUnit{UnitID: 7, Command: 4},
		&PetCommandPoint{X: 100, Y: 200}, &TakeWaypoint{ObjectID: 0x0a0b0c0d, Dest: 39},
		&ClaimUnit{UnitType: 1, UnitID: 9}, &FinishConvert{ItemID: 0xffffffff}, &NpcIntroFlag{MonsterClass: 150},
		&ClickButton{Button: 1, Arg1: 2, Arg2: 3}, &DropGold{PlayerID: 4, Amount: 5000},
		&GiveItemToMerc{BodyLoc: 3}, &ReviveMerc{MercID: 8}, &PickItemToBelt{ItemID: 6}, &BareCommand{ID: 0x41},
	} {
		wire := m.MarshalPacket()

		got, err := DecodeClient(wire)
		if err != nil {
			t.Fatalf("%T: %v", m, err)
		}

		if !reflect.DeepEqual(got, Message(m)) {
			t.Errorf("%T: got %+v want %+v", m, got, m)
		}
	}

	// field layout pins read from the handlers
	if w := (TakeWaypoint{ObjectID: 0x01020304, Dest: 0x0a0b}).MarshalPacket(); !bytes.Equal(w, []byte{0x49, 4, 3, 2, 1, 0x0b, 0x0a, 0, 0}) {
		t.Errorf("TakeWaypoint % x", w)
	}

	if w := (PetCommandPoint{X: 0x0102, Y: 0x0304}).MarshalPacket(); w[5] != 2 || w[9] != 4 {
		t.Errorf("PetCommandPoint % x", w)
	}
}

func TestPacketNamesUpdated(t *testing.T) {
	if n, ok := PacketName(0x49, ClientToServer); n != "TakeWaypoint" || !ok {
		t.Errorf("%q %v", n, ok)
	}

	if n, ok := PacketName(0x73, ServerToClient); n != "MissileCreate" || !ok {
		t.Errorf("%q %v", n, ok)
	}
}
