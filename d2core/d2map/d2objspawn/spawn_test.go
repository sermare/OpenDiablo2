package d2objspawn

import (
	"errors"
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2object"
	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2records"
)

const objTxt = "Name\tSpawnMax\tTrapProb\tAct\tSubClass\tOperateFn\tPopulateFn\tInitFn\tOperateRange\tLockable\tRestoreVirgins\tParm0\tDamage\tSelectable0\n" +
	"Dummy\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\n" +
	"chest\t5\t0\t1\t8\t4\t3\t3\t2\t1\t1\t18\t0\t1\n" +
	"Waypoint\t0\t0\t1\t64\t23\t0\t17\t2\t0\t0\t0\t0\t1\n" +
	"act4only\t0\t0\t8\t8\t4\t3\t3\t2\t1\t1\t18\t0\t1\n"

func groupTxt() []byte {
	hdr := []string{"GroupName", "Offset"}
	row := []string{"G", "1"}

	for i := 0; i < 8; i++ {
		s := string(rune('0' + i))
		hdr = append(hdr, "ID"+s, "DENSITY"+s, "PROB"+s)

		if i == 0 {
			row = append(row, "1", "100", "100")
		} else {
			row = append(row, "0", "0", "0")
		}
	}

	hdr = append(hdr, "SHRINES", "WELLS")
	row = append(row, "0", "0")

	return []byte(strings.Join(hdr, "\t") + "\n" + strings.Join(row, "\t") + "\n")
}

type fakeRoom struct {
	tiles  int
	placed []int
}

func (f *fakeRoom) Tiles() int   { return f.tiles }
func (f *fakeRoom) Place(id int) { f.placed = append(f.placed, id) }

func tables(t *testing.T) Tables {
	t.Helper()

	tb, err := FromText([]byte(objTxt), groupTxt())
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func TestPopulateRoom(t *testing.T) {
	tb := tables(t)
	lv := Level{Act: 1, Groups: [8]int{1}, Probs: [8]int{100}}

	room := &fakeRoom{tiles: 200}
	n := PopulateRoom(tb, lv, false, room, 7)

	if n == 0 || n != len(room.placed) || n > 5 {
		t.Fatalf("placed %d %v (SpawnMax 5)", n, room.placed)
	}

	for _, id := range room.placed {
		if id != 1 {
			t.Errorf("unexpected object %d", id)
		}
	}

	again := &fakeRoom{tiles: 200}
	PopulateRoom(tb, lv, false, again, 7)

	if len(again.placed) != len(room.placed) {
		t.Error("not deterministic")
	}

	// act 4 level cannot place the act 1 chest
	if PopulateRoom(tb, Level{Act: 4, Groups: [8]int{1}, Probs: [8]int{100}}, false, &fakeRoom{tiles: 200}, 7) != 0 {
		t.Error("act mask ignored")
	}

	// no group chosen
	if PopulateRoom(tb, Level{Act: 1}, false, &fakeRoom{tiles: 200}, 7) != 0 {
		t.Error("empty level placed objects")
	}

	// unknown group id
	if PopulateRoom(tb, Level{Act: 1, Groups: [8]int{9}, Probs: [8]int{100}}, false, &fakeRoom{tiles: 200}, 7) != 0 {
		t.Error("unknown group placed objects")
	}
}

func TestWaypointBit(t *testing.T) {
	tb := tables(t)

	if bit, ok := tb.WaypointBit(2, 3, false); !ok || bit != 1 {
		t.Errorf("cold plains bit = %d %v", bit, ok)
	}

	if _, ok := tb.WaypointBit(1, 3, false); ok {
		t.Error("chest is not a waypoint")
	}

	if _, ok := tb.WaypointBit(99, 3, false); ok {
		t.Error("unknown id")
	}
}

func TestOpenChest(t *testing.T) {
	tb := tables(t)
	st := d2object.NewChest(tb.Defs[1], d2object.NewRoller(1))
	drops := 0
	h := d2object.ChestHooks{
		TreasureClass: func(d2object.Def, int, int, int) string { return "Act 1 Chest A" },
		Drop:          func(string, int, uint32) int { drops++; return 3 },
	}

	res, err := tb.OpenChest(1, &st, h, 1, 0, 5, 9)
	if err != nil || res.Dropped != 3 || res.TreasureClass != "Act 1 Chest A" || drops != 1 {
		t.Fatalf("%+v %v", res, err)
	}

	if _, err := tb.OpenChest(1, &st, h, 1, 0, 5, 9); !errors.Is(err, d2object.ErrAlreadyOpened) {
		t.Errorf("second open: %v", err)
	}

	if _, err := tb.OpenChest(50, &st, h, 1, 0, 5, 9); !errors.Is(err, ErrUnknownObject) {
		t.Errorf("unknown: %v", err)
	}
}

func TestFromRecords(t *testing.T) {
	objs := d2records.ObjectDetails{
		2: {Name: "Waypoint", Act: 1, SubClass: 64, OperateFn: 23},
	}
	m := [8]d2records.ObjectGroupMember{{ID: 2, Density: 5, Probability: 50}}
	groups := d2records.ObjectGroups{4: {GroupName: "g", Offset: 4, Members: &m}}

	tb := FromRecords(objs, groups)
	if len(tb.Defs) != 3 || tb.Defs[2].Name != "Waypoint" || tb.Groups[4].Members[0].Prob != 50 {
		t.Fatalf("%+v", tb)
	}

	if _, ok := tb.WaypointBit(2, 3, false); !ok {
		t.Error("record-built waypoint not found")
	}
}
