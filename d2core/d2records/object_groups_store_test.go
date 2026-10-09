package d2records

import (
	"strings"
	"testing"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2txt"
	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2util"
)

// The loader must store its records on the manager (it used to drop them).
func TestObjectGroupsLoaderStoresRecords(t *testing.T) {
	hdr := []string{"GroupName", "Offset"}
	row := []string{"Test", "3"}

	for i := 0; i < 8; i++ {
		s := string(rune('0' + i))
		hdr = append(hdr, "ID"+s, "DENSITY"+s, "PROB"+s)
		row = append(row, "5", "10", "20")
	}

	hdr = append(hdr, "SHRINES", "WELLS")
	row = append(row, "0", "0")

	d := d2txt.LoadDataDictionary([]byte(strings.Join(hdr, "\t") + "\n" + strings.Join(row, "\t") + "\n"))

	r := &RecordManager{Logger: d2util.NewLogger()}
	if err := objectGroupsLoader(r, d); err != nil {
		t.Fatal(err)
	}

	g := r.Object.Groups[3]
	if g == nil || g.GroupName != "Test" || g.Members[7].Density != 10 {
		t.Fatalf("group not stored: %+v", r.Object.Groups)
	}
}
