package d2s

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"testing"
)

func realSample(t *testing.T) ([]byte, *ItemTables) {
	t.Helper()

	path := os.Getenv("D2S_SAMPLE_BODY")
	if path == "" || os.Getenv("D2_TABLES") == "" {
		t.Skip("set D2_TABLES and D2S_SAMPLE_BODY to run")
	}

	tables := loadRealTables(t, false)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return data, tables
}

func firstDiff(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}

	return -1
}

// refix makes the size field and checksum of a (possibly damaged) save valid
// again so that the parser reaches the damage.
func refix(d []byte) {
	if len(d) < checksumOffset+checksumLength {
		return
	}

	binary.LittleEndian.PutUint32(d[8:], uint32(len(d)))
	binary.LittleEndian.PutUint32(d[checksumOffset:], 0)
	binary.LittleEndian.PutUint32(d[checksumOffset:], Checksum(d))
}

func TestWriteRealRoundTrip(t *testing.T) {
	data, tables := realSample(t)

	c, err := Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Write(c, tables)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatalf("round trip differs: len %d vs %d, first diff at 0x%X", len(out), len(data), firstDiff(out, data))
	}
}

func TestWriteMutation(t *testing.T) {
	data, tables := realSample(t)

	c, err := Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	c.Body.SetStat(StatGold, 123456)

	idx := -1

	for i, it := range c.Items {
		if it.MaxDurability > 0 && it.Durability > 1 {
			idx = i
			break
		}
	}

	if idx < 0 {
		t.Fatal("no item with durability")
	}

	want := c.Items[idx].Durability - 1
	c.Items[idx].Durability = want

	out, err := Write(c, tables)
	if err != nil {
		t.Fatal(err)
	}

	if bytes.Equal(out, data) {
		t.Fatal("mutation had no effect")
	}

	back, err := Parse(out, tables) // also validates size and checksum
	if err != nil {
		t.Fatal(err)
	}

	if back.Body.Attributes.Gold != 123456 || back.Items[idx].Durability != want {
		t.Fatalf("gold=%d durability=%d want %d", back.Body.Attributes.Gold, back.Items[idx].Durability, want)
	}

	if len(back.Items) != len(c.Items) {
		t.Fatal("item count changed")
	}
}

func TestWriteRealErrors(t *testing.T) {
	data, tables := realSample(t)

	c, err := Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	// unknown class: Write refuses it, and so does Parse
	c.Header.Class = Assassin + 1
	if _, err = Write(c, tables); !errors.Is(err, ErrInvalidClass) {
		t.Fatalf("write unknown class: got %v", err)
	}

	bad := append([]byte(nil), data...)
	bad[classOffset] = 99
	refix(bad)

	if _, err = Parse(bad, tables); !errors.Is(err, ErrInvalidClass) {
		t.Fatalf("parse unknown class: got %v", err)
	}

	// truncated inputs are rejected (the last 3 bytes are an optional golem
	// marker that the parser tolerates missing)
	for _, n := range []int{10, HeaderSize - 1, HeaderSize + 100, len(data) / 2, len(data) - 6} {
		cut := append([]byte(nil), data[:n]...)
		refix(cut)

		if _, err = Parse(cut, tables); err == nil {
			t.Fatalf("truncated to %d bytes: expected an error", n)
		}
	}

	c.Header.Class = Sorceress

	// a body is required, and so are the tables
	if _, err = Write(&Character{Header: c.Header}, tables); !errors.Is(err, ErrNoBody) {
		t.Fatalf("no body: got %v", err)
	}

	if _, err = Write(c, nil); !errors.Is(err, ErrNoTables) {
		t.Fatalf("no tables: got %v", err)
	}

	// SocketCount must agree with Children
	c.Items[0].SocketCount, c.Items[0].Simple = 3, false
	c.Items[0].Children = nil

	if _, err = Write(c, tables); !errors.Is(err, ErrSocketCount) {
		t.Fatalf("socket count: got %v", err)
	}
}

func TestWriteNewCharacterRoundTrip(t *testing.T) {
	data := buildSave("Fresh", Paladin, 1, StatusNewCharacter|StatusExpansion)

	c, err := Parse(data, nil)
	if err != nil {
		t.Fatal(err)
	}

	out, err := Write(c, nil)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(out, data) {
		t.Fatal("new character round trip differs")
	}

	c.Header.Name = "Renamed"

	out, err = Write(c, nil)
	if err != nil {
		t.Fatal(err)
	}

	h, err := ParseHeader(out)
	if err != nil || h.Name != "Renamed" {
		t.Fatalf("rename: %v %+v", err, h)
	}

	c.Header.Name = "WayTooLongCharacterName"
	if _, err = Write(c, nil); !errors.Is(err, ErrBadName) {
		t.Fatalf("long name: got %v", err)
	}
}

// syntheticFull builds a complete expansion save from buildBody output with
// the given top-level item bytes, an empty corpse list and no mercenary.
func syntheticFull(t *testing.T, items []byte, count int) []byte {
	t.Helper()

	data := buildBody(t, map[int]uint64{StatStrength: 30}, [numSkills]byte{1, 2, 3})
	data = data[:len(data)-4] // drop the empty 'JM' list
	data = append(data, 'J', 'M', byte(count), 0)
	data = append(data, items...)
	data = append(data, 'J', 'M', 0, 0) // corpse
	data = append(data, 'j', 'f')
	refix(data)

	return data
}

func syntheticItems() []byte {
	// a simple item, a stackable key with a parameterised property, and an ear
	w := &bitWriter{}
	writeItemHeader(w, 1<<4|1<<21, 0, 0, 3, 2, 1)
	writeCode(w, "rvs")
	w.write(0, 3)
	out := append([]byte(nil), w.bytes()...)

	w = &bitWriter{}
	writeItemHeader(w, 1<<4, 0, 0, 0, 0, 0)
	writeCode(w, "key")
	w.write(0, 3)
	w.write(0xDEADBEEF, 32)
	w.write(30, 7)
	w.write(2, 4)
	w.write(0, 1)
	w.write(0, 1)
	w.write(1, 1)   // timestamp
	w.write(12, 9)  // quantity
	w.write(97, 9)  // stat id
	w.write(300, 9) // param
	w.write(7, 6)   // value
	w.write(0x1FF, 9)
	out = append(out, w.bytes()...)

	w = &bitWriter{}
	writeItemHeader(w, 1<<16, 0, 0, 0, 0, 0)
	w.write(2, 3)
	w.write(40, 7)

	for _, c := range "Bob" {
		w.write(uint64(c), 7)
	}

	w.write(0, 7)

	return append(out, w.bytes()...)
}

func TestWriteSyntheticRoundTrip(t *testing.T) {
	tb := miniTables(t)

	for name, data := range map[string][]byte{
		"empty":      syntheticFull(t, nil, 0),
		"with items": syntheticFull(t, syntheticItems(), 3),
	} {
		c, err := Parse(data, tb)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		out, err := Write(c, tb)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}

		if !bytes.Equal(out, data) {
			t.Fatalf("%s: round trip differs at 0x%X (len %d vs %d)", name, firstDiff(out, data), len(out), len(data))
		}
	}

	// mutate a quantity and a core stat in the synthetic save
	c, err := Parse(syntheticFull(t, syntheticItems(), 3), tb)
	if err != nil {
		t.Fatal(err)
	}

	c.Items[1].Quantity = 99
	c.Body.SetStat(StatStrength, 55)

	out, err := Write(c, tb)
	if err != nil {
		t.Fatal(err)
	}

	back, err := Parse(out, tb)
	if err != nil {
		t.Fatal(err)
	}

	if back.Items[1].Quantity != 99 || back.Body.Attributes.Strength != 55 {
		t.Fatalf("mutation lost: %+v %+v", back.Items[1], back.Body.Attributes)
	}
}

func TestWriteItemErrors(t *testing.T) {
	tb := miniTables(t)
	it := Item{Code: "zzz", Quality: QualityNormal}

	if _, err := marshalItem(&it, tb); !errors.Is(err, ErrUnknownItem) {
		t.Fatalf("unknown code: %v", err)
	}

	it = Item{Code: "key", Quality: QualityNormal, Quantity: 1 << 9}
	if _, err := marshalItem(&it, tb); !errors.Is(err, ErrValueRange) {
		t.Fatalf("overflow: %v", err)
	}

	it = Item{Code: "key", Quality: QualityNormal, Properties: []Property{{ID: 17}}}
	if _, err := marshalItem(&it, tb); err == nil {
		t.Fatal("expected an error for a stat the tables do not know")
	}
}
