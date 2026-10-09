package d2records

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
)

// objgroup.txt has 8 member slots (ID0..ID7), see d2object.GroupMembers.
func TestCreateMembersReadsEightSlots(t *testing.T) {
	hdr := []string{"GroupName", "Offset"}
	row := []string{"Test", "1"}

	for i := 0; i < 8; i++ {
		s := string(rune('0' + i))
		hdr = append(hdr, "ID"+s, "DENSITY"+s, "PROB"+s)
		row = append(row, s, "10", "20")
	}

	hdr = append(hdr, "SHRINES", "WELLS")
	row = append(row, "0", "0")

	d := d2txt.LoadDataDictionary([]byte(strings.Join(hdr, "\t") + "\n" + strings.Join(row, "\t") + "\n"))
	if !d.Next() {
		t.Fatal("no row")
	}

	m := createMembers(d, false)
	if len(m) != 8 {
		t.Fatalf("members %d", len(m))
	}

	if m[7].ID != 7 || m[7].Density != 10 || m[7].Probability != 20 {
		t.Errorf("slot 7 not read: %+v", m[7])
	}
}
