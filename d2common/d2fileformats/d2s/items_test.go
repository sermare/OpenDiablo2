package d2s

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// bitWriter builds LSB-first bit streams for hand-encoded test items.
type bitWriter struct {
	buf []byte
	pos int
}

func (w *bitWriter) write(v uint64, n int) {
	for i := 0; i < n; i++ {
		if w.pos/8 >= len(w.buf) {
			w.buf = append(w.buf, 0)
		}

		if v>>uint(i)&1 == 1 {
			w.buf[w.pos/8] |= 1 << uint(w.pos%8)
		}

		w.pos++
	}
}

func (w *bitWriter) bytes() []byte { return w.buf }

func TestBitReader(t *testing.T) {
	r := newBitReader([]byte{0xA5, 0x0F, 0xFF})

	// 0xA5 = 1010 0101: LSB first.
	if v, _ := r.read(4); v != 0x5 {
		t.Fatalf("low nibble = %#x", v)
	}

	if v, _ := r.read(8); v != 0xFA { // high nibble of 0xA5 then low of 0x0F
		t.Fatalf("straddling read = %#x", v)
	}

	r.align()

	if r.pos != 16 {
		t.Fatalf("align pos = %d", r.pos)
	}

	if v, _ := r.read(8); v != 0xFF {
		t.Fatalf("last byte = %#x", v)
	}

	if _, err := r.read(1); err != ErrUnexpectedEOF {
		t.Fatalf("expected EOF, got %v", err)
	}
}

const (
	miniStatCost = "Stat\tID\tSigned\tEncode\tSave Bits\tSave Add\tSave Param Bits\tCSvBits\tCSvParam\n" +
		"strength\t0\t1\t0\t8\t32\t\t10\t\n" +
		"armorclass\t31\t1\t0\t11\t10\t\t\t\n" +
		"durability\t72\t0\t0\t9\t0\t\t\t\n" +
		"maxdurability\t73\t0\t0\t8\t0\t\t\t\n" +
		"item_nonclassskill\t97\t0\t1\t6\t0\t9\t\t\n"
	miniItemTypes = "ItemType\tCode\tEquiv1\tEquiv2\nBook\tbook\tmisc\t\nMisc\tmisc\t\t\nKey\tkey\tmisc\t\n"
	miniArmor     = "name\tcode\ttype\tstackable\tnodurability\nCap\tcap\thelm\t0\t0\n"
	miniWeapons   = "name\tcode\ttype\tstackable\tnodurability\nBow\tbow\tbow\t0\t0\n"
	miniMisc      = "name\tcode\ttype\tstackable\tnodurability\nKey\tkey\tkey\t1\t1\nTome\ttbk\tbook\t1\t1\n"
)

func miniTables(t *testing.T) *ItemTables {
	t.Helper()

	tb, err := NewItemTables([]byte(miniStatCost), []byte(miniArmor), []byte(miniWeapons),
		[]byte(miniMisc), []byte(miniItemTypes))
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

// writeItemHeader writes 'JM', flags and the location block.
func writeItemHeader(w *bitWriter, flags uint32, loc, eq, x, y, page uint64) {
	w.write('J'|'M'<<8, 16)
	w.write(uint64(flags), 32)
	w.write(0x65, 10) // version
	w.write(loc, 3)
	w.write(eq, 4)
	w.write(x, 4)
	w.write(y, 4)
	w.write(page, 3)
}

func writeCode(w *bitWriter, code string) {
	for i := 0; i < 4; i++ {
		c := byte(' ')
		if i < len(code) {
			c = code[i]
		}

		w.write(uint64(c), 8)
	}
}

func TestParseItemListHandEncoded(t *testing.T) {
	tb := miniTables(t)

	// Item 1: a simple item (flag bits: identified, simple).
	w := &bitWriter{}
	writeItemHeader(w, 1<<4|1<<21, 0, 0, 3, 2, 1)
	writeCode(w, "rvs")
	w.write(0, 3)
	simple := append([]byte(nil), w.bytes()...)

	// Item 2: a stackable extended misc item (key), normal quality, one
	// property (id 97 with a 9 bit param) and the list terminator. Misc
	// items have no defense or durability fields.
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
	key := w.bytes()

	// Item 3: an ear.
	w = &bitWriter{}
	writeItemHeader(w, 1<<16, 0, 0, 0, 0, 0)
	w.write(2, 3)
	w.write(40, 7)

	for _, c := range "Bob" {
		w.write(uint64(c), 7)
	}

	w.write(0, 7)
	ear := w.bytes()

	list := []byte{'J', 'M', 3, 0}
	list = append(list, simple...)
	list = append(list, key...)
	list = append(list, ear...)
	list = append(list, 0xEE, 0xEE) // trailing bytes that must not be consumed

	items, n, err := ParseItemList(list, tb)
	if err != nil {
		t.Fatal(err)
	}

	if n != len(list)-2 {
		t.Fatalf("consumed %d, want %d", n, len(list)-2)
	}

	if len(items) != 3 {
		t.Fatalf("got %d items", len(items))
	}

	s := items[0]
	if s.Code != "rvs" || !s.Simple || !s.Identified || s.X != 3 || s.Y != 2 || s.Page != 1 || s.Version != 0x65 {
		t.Errorf("simple item decoded wrong: %+v", s)
	}

	k := items[1]
	if k.Code != "key" || k.ID != 0xDEADBEEF || k.Level != 30 || k.Quality != QualityNormal ||
		k.Quantity != 12 || !k.Timestamp {
		t.Errorf("key decoded wrong: %+v", k)
	}

	if len(k.Properties) != 1 || k.Properties[0] != (Property{ID: 97, Name: "item_nonclassskill", Param: 300, Value: 7}) {
		t.Errorf("key properties = %+v", k.Properties)
	}

	if e := items[2]; e.EarInfo == nil || e.EarInfo.Name != "Bob" || e.EarInfo.Level != 40 || e.EarInfo.Class != 2 {
		t.Errorf("ear decoded wrong: %+v", e)
	}
}

func TestParseItemListErrors(t *testing.T) {
	tb := miniTables(t)

	if _, _, err := ParseItemList([]byte{'X', 'Y', 0, 0}, tb); err != ErrNoItemList {
		t.Errorf("bad tag: %v", err)
	}

	if _, _, err := ParseItemList([]byte{'J', 'M', 1, 0, 'J', 'M'}, tb); err == nil {
		t.Error("truncated item should fail")
	}

	if _, _, err := ParseItemList([]byte{'J', 'M', 0, 0}, nil); err != ErrNoTables {
		t.Errorf("nil tables: %v", err)
	}
}

// ---- real data tests, enabled by environment variables ----

func loadRealTables(t *testing.T, preferTxt bool) *ItemTables {
	t.Helper()

	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	read := func(names ...string) []byte {
		for _, n := range names {
			if b, err := os.ReadFile(filepath.Join(dir, n)); err == nil {
				return b
			}
		}

		t.Skipf("none of %v found in %s", names, dir)

		return nil
	}

	var isc []byte
	if preferTxt {
		isc = read("ItemStatCost.txt")
	} else {
		isc = read("itemstatcost.bin", "ItemStatCost.txt")
	}

	tb, err := NewItemTables(isc, read("armor.txt"), read("weapons.txt"), read("misc.txt"), read("ItemTypes.txt"))
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func TestStatTablesBinMatchesText(t *testing.T) {
	dir := os.Getenv("D2_TABLES")
	if dir == "" {
		t.Skip("D2_TABLES not set")
	}

	bin, err1 := os.ReadFile(filepath.Join(dir, "itemstatcost.bin"))
	txt, err2 := os.ReadFile(filepath.Join(dir, "ItemStatCost.txt"))

	if err1 != nil || err2 != nil {
		t.Skip("need both itemstatcost.bin and ItemStatCost.txt")
	}

	tb := &ItemTables{stats: map[int]StatInfo{}}
	tt := &ItemTables{stats: map[int]StatInfo{}}

	if err := tb.loadStatBin(bin); err != nil {
		t.Fatal(err)
	}

	if err := tt.loadStatText(txt); err != nil {
		t.Fatal(err)
	}

	for id, a := range tt.stats {
		b := tb.stats[id]
		a.Name = ""

		if a != b {
			t.Errorf("stat %d: txt %+v, bin %+v", id, a, b)
		}
	}

	// Character section widths (see also the lead's stats parser).
	want := map[int]int{0: 10, 1: 10, 2: 10, 3: 10, 4: 10, 5: 8, 6: 21, 7: 21, 8: 21, 9: 21,
		10: 21, 11: 21, 12: 7, 13: 32, 14: 25, 15: 25}
	for id, w := range want {
		if got, _, ok := tb.CharStatInfo(id); !ok || got != w {
			t.Errorf("CharStatInfo(%d) = %d,%v want %d", id, got, ok, w)
		}
	}
}

type refAttr struct {
	ID     int     `json:"id"`
	Values []int64 `json:"values"`
}

type refItem struct {
	Type       string      `json:"type"`
	Quality    uint8       `json:"quality"`
	Location   uint8       `json:"location_id"`
	Equipped   uint8       `json:"equipped_id"`
	PosX       uint8       `json:"position_x"`
	PosY       uint8       `json:"position_y"`
	Simple     uint8       `json:"simple_item"`
	Ethereal   uint8       `json:"ethereal"`
	Level      uint8       `json:"level"`
	ItemID     uint32      `json:"id"`
	Quantity   uint16      `json:"quantity"`
	Defense    int         `json:"defense_rating"`
	MaxDur     uint16      `json:"max_durability"`
	CurDur     uint16      `json:"current_durability"`
	Sockets    uint8       `json:"total_nr_of_sockets"`
	Unique     uint16      `json:"unique_id"`
	SetID      uint16      `json:"set_id"`
	Prefix     uint16      `json:"magic_prefix"`
	Suffix     uint16      `json:"magic_suffix"`
	Runeword   uint16      `json:"runeword_id"`
	Attrs      []refAttr   `json:"magic_attributes"`
	SetAttrs   [][]refAttr `json:"set_attributes"`
	RuneAttrs  []refAttr   `json:"runeword_attributes"`
	Socketed   []refItem   `json:"socketed_items"`
	NrSocketed uint8       `json:"nr_of_items_in_sockets"`
}

// findItemList locates the item list in a save (or save body): the first
// 'JM' that directly follows the 30 byte skill block after the 'if' tag.
func findItemList(t *testing.T, data []byte) int {
	t.Helper()

	gf := bytes.Index(data, []byte("gf"))
	if gf < 0 {
		t.Fatal("no gf section")
	}

	for off := gf; ; {
		i := bytes.Index(data[off:], []byte("if"))
		if i < 0 {
			t.Fatal("no if/JM sequence")
		}

		p := off + i
		if p+34 <= len(data) && data[p+32] == 'J' && data[p+33] == 'M' {
			return p + 32
		}

		off = p + 1
	}
}

// flatten converts parsed properties to the reference's per-id value lists.
func flatten(tb *ItemTables, props []Property) []refAttr {
	var out []refAttr

	for i := 0; i < len(props); i++ {
		p := props[i]
		a := refAttr{ID: p.ID}

		st, _ := tb.Stat(p.ID)

		switch {
		case st.Encode == 2 || st.Encode == 3: // skill level (6 bits) and skill id (10 bits)
			a.Values = append(a.Values, int64(p.Param&63), int64(p.Param>>6))
		case p.ID == 188: // skill tab: 3 bit tab, 13 bit class/level part
			a.Values = append(a.Values, int64(p.Param&7), int64(p.Param>>3))
		case st.SaveParamBits > 0:
			a.Values = append(a.Values, int64(p.Param))
		}

		if st.Encode == 3 { // charges: current in the low byte, maximum in the high byte
			a.Values = append(a.Values, p.Value&0xFF, p.Value>>8)
		} else {
			a.Values = append(a.Values, p.Value)
		}

		for _, f := range groupFollowers[p.ID] {
			i++
			a.Values = append(a.Values, props[i].Value)
			_ = f
		}

		out = append(out, a)
	}

	return out
}

func compareAttrs(t *testing.T, where string, tb *ItemTables, got []Property, want []refAttr) {
	t.Helper()

	g := flatten(tb, got)
	if len(g) != len(want) {
		t.Errorf("%s: %d properties, want %d (%v vs %v)", where, len(g), len(want), g, want)
		return
	}

	for i := range g {
		if g[i].ID != want[i].ID || !equalInts(g[i].Values, want[i].Values) {
			t.Errorf("%s prop %d: got %+v want %+v", where, i, g[i], want[i])
		}
	}
}

func equalInts(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func compareItem(t *testing.T, where string, tb *ItemTables, got *Item, want *refItem) {
	t.Helper()

	if got.Code != want.Type || got.Quality != want.Quality && !got.Simple ||
		got.Location != want.Location || got.Equipped != want.Equipped ||
		got.X != want.PosX || got.Y != want.PosY || got.Simple != (want.Simple == 1) {
		t.Errorf("%s: header mismatch: got %s q%d loc%d eq%d (%d,%d) simple=%v, want %+v",
			where, got.Code, got.Quality, got.Location, got.Equipped, got.X, got.Y, got.Simple, want)
		return
	}

	if got.Simple {
		return
	}

	if got.Level != want.Level || got.ID != want.ItemID || got.Quantity != want.Quantity ||
		got.Defense != want.Defense || got.MaxDurability != want.MaxDur ||
		got.Durability != want.CurDur || got.TotalSockets != want.Sockets ||
		got.UniqueID != want.Unique || got.SetID != want.SetID ||
		got.MagicPrefix != want.Prefix || got.MagicSuffix != want.Suffix ||
		got.RunewordID != want.Runeword || got.Ethereal != (want.Ethereal == 1) {
		t.Errorf("%s: field mismatch:\n got %+v\nwant %+v", where, got, want)
	}

	compareAttrs(t, where+" props", tb, got.Properties, want.Attrs)
	compareAttrs(t, where+" runeword props", tb, got.RunewordProperties, want.RuneAttrs)

	if len(got.SetProperties) != len(want.SetAttrs) {
		t.Errorf("%s: %d set lists, want %d", where, len(got.SetProperties), len(want.SetAttrs))
	} else {
		for i := range got.SetProperties {
			compareAttrs(t, where+" set list", tb, got.SetProperties[i], want.SetAttrs[i])
		}
	}

	if len(got.Children) != len(want.Socketed) {
		t.Errorf("%s: %d socketed items, want %d", where, len(got.Children), len(want.Socketed))
		return
	}

	for i := range got.Children {
		compareItem(t, where+" socket", tb, &got.Children[i], &want.Socketed[i])
	}
}

func TestParseItemListAgainstReference(t *testing.T) {
	path := os.Getenv("D2S_SAMPLE_BODY")
	if path == "" {
		t.Skip("D2S_SAMPLE_BODY not set")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	jsonPath := os.Getenv("D2S_SAMPLE_JSON")
	if jsonPath == "" {
		jsonPath = path + ".json"
	}

	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Skipf("no reference json: %v", err)
	}

	var ref struct {
		Items []refItem `json:"items"`
	}
	if err = json.Unmarshal(raw, &ref); err != nil {
		t.Fatal(err)
	}

	start := findItemList(t, data)

	for _, preferTxt := range []bool{false, true} {
		tb := loadRealTables(t, preferTxt)

		items, n, perr := ParseItemList(data[start:], tb)
		if perr != nil {
			t.Fatalf("txt=%v: %v", preferTxt, perr)
		}

		if len(items) != 60 || len(ref.Items) != 60 {
			t.Fatalf("got %d items, reference has %d, want 60", len(items), len(ref.Items))
		}

		for i := range items {
			compareItem(t, "item "+itoa(i), tb, &items[i], &ref.Items[i])
		}

		// The corpse list ('JM' + count) must follow immediately.
		if rest := data[start+n:]; len(rest) < 4 || rest[0] != 'J' || rest[1] != 'M' {
			t.Errorf("no JM tag after the item list, got % x", rest)
		}
	}
}

func itoa(i int) string {
	b, _ := json.Marshal(i)
	return string(b)
}

// TestParseFullCharacter parses a whole real save (header, body, every item
// section) and checks it against the reference output when the env vars are set.
func TestParseFullCharacter(t *testing.T) {
	dir, path := os.Getenv("D2_TABLES"), os.Getenv("D2S_SAMPLE_BODY")
	if dir == "" || path == "" {
		t.Skip("set D2_TABLES and D2S_SAMPLE_BODY to run")
	}

	tables := loadRealTables(t, false)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	c, err := Parse(data, tables)
	if err != nil {
		t.Fatal(err)
	}

	if c.Header.Name != "NokkaSorc" || c.Body.Attributes.Level != 94 || len(c.Items) != 60 {
		t.Fatalf("name=%q level=%d items=%d", c.Header.Name, c.Body.Attributes.Level, len(c.Items))
	}

	if c.HasCorpse || len(c.Corpse) != 0 || len(c.MercItems) != 0 || c.Golem != nil {
		t.Fatalf("unexpected corpse/merc/golem: %+v", c)
	}

	m := c.Header.Mercenary
	if m.ID != 0xB43B75AF || m.NameID != 7 || m.Type != 11 || m.Experience != 100730580 || m.Dead {
		t.Fatalf("mercenary: %+v", m)
	}
}
