package d2unitmsg

import (
	"reflect"
	"testing"
)

func TestFlushOrder(t *testing.T) {
	d := NewDirty()
	a := &Unit{Type: 1, ID: 10}
	b := &Unit{Type: 1, ID: 20}

	d.QueueCastSkillAtUnit(a, 7, 99)
	d.QueueSUnit(b, 3)
	d.QueueSUnit(a, 4) // a keeps its first place in the dirty list

	if a.Pending() != 2 || b.Pending() != 1 {
		t.Fatal("pending counts")
	}

	var got []byte

	var ids []uint32

	d.Flush(func(u *Unit, m Message) { got = append(got, m.Type); ids = append(ids, u.ID) })

	if !reflect.DeepEqual(got, []byte{TypeCastSkillAtUnit, TypeSUnit, TypeSUnit}) || !reflect.DeepEqual(ids, []uint32{10, 10, 20}) {
		t.Fatalf("%x %v", got, ids)
	}

	if a.Pending() != 0 {
		t.Fatal("drained")
	}

	d.Flush(func(*Unit, Message) { t.Fatal("nothing left") })
}

func TestSUnitClearsState(t *testing.T) {
	d := NewDirty()
	u := &Unit{Type: 1, ID: 5, States: map[int]bool{StateClearedBySUnit: true, 3: true}}
	d.QueueSUnit(u, 1)

	if u.States[StateClearedBySUnit] || !u.States[3] {
		t.Fatalf("%v", u.States)
	}

	d.QueueCastSkillAtUnit(u, 1, 1)

	u.States[StateClearedBySUnit] = true
	d.Queue(u, Message{Type: Type23})

	if !u.States[StateClearedBySUnit] {
		t.Fatal("only 0xa5 clears the state")
	}
}

func TestNilUnitIdentity(t *testing.T) {
	if ty, id := ident(nil); ty != NoUnit || id != NoID {
		t.Fatal("null unit is type 6 id -1")
	}
}

func TestMercStatLimit(t *testing.T) {
	d := NewDirty()
	u := &Unit{}

	tests := []struct {
		stat int
		ok   bool
	}{{0, true}, {0xfe, true}, {0xff, false}, {-1, false}}
	for _, tc := range tests {
		if d.QueueMercStat(u, tc.stat, 1) != tc.ok {
			t.Errorf("stat %d", tc.stat)
		}
	}

	if u.Pending() != 2 {
		t.Fatalf("%d queued", u.Pending())
	}
}

func TestPurgeAttacker(t *testing.T) {
	var l HitList

	l.Add(Hit{1, 10, 0})
	l.Add(Hit{1, 11, 1})
	l.Add(Hit{1, 10, 2})
	l.Add(Hit{0, 10, 3}) // same id, other type: stays
	l.Add(Hit{1, 12, 4})

	if n := l.PurgeAttacker(1, 10); n != 2 {
		t.Fatalf("removed %d", n)
	}

	want := []Hit{{1, 11, 1}, {0, 10, 3}, {1, 12, 4}}
	if !reflect.DeepEqual(l.Entries(), want) {
		t.Fatalf("%v", l.Entries())
	}

	if l.PurgeAttacker(9, 9) != 0 || l.Len() != 3 {
		t.Fatal("no match, no change")
	}

	var empty HitList
	if empty.PurgeAttacker(1, 1) != 0 {
		t.Fatal("empty list")
	}
}
