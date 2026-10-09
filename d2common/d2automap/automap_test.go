package d2automap

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const synthTxt = "LevelName\tTileName\tStyle\tStartSequence\tEndSequence\tType1\tCel1\tType2\tCel2\tType2\tCel3\tType4\tCel4\n" +
	"1 Town\tfl\t0\t1\t46\tFloor A\t0\tFloor B\t1\tFloor C\t2\tFloor D\t3\n" +
	"1 Town\tfl\t2\t4\t7\tRiver\t4\t\t-1\t\t-1\t\t-1\n" +
	"1 Town\twl\t-1\t-1\t-1\tany wall\t30\t\t-1\t\t-1\t\t-1\n" +
	"1 Cave\tfl\t0\t0\t3\tcave\t60\t\t-1\t\t-1\t\t-1\n" +
	"Expansion\n"

// blockTxt has "1 Town" in two blocks; the engine keeps the last one.
const blockTxt = "LevelName\tTileName\tStyle\tStartSequence\tEndSequence\tType1\tCel1\tType2\tCel2\tType2\tCel3\tType4\tCel4\n" +
	"1 Town\tfl\t0\t1\t46\ta\t0\t\t-1\t\t-1\t\t-1\n" +
	"1 Cave\tfl\t0\t0\t3\tb\t60\t\t-1\t\t-1\t\t-1\n" +
	"Expansion\n" +
	"1 Town\tfl\t9\t0\t0\tlate\t99\t\t-1\t\t-1\t\t-1\n"

// synthBin builds an automap.bin for the same rows as synthTxt (without the marker).
func synthBin() []byte {
	type r struct {
		lvl, tile string
		st, a, b  byte
		cels      [4]int32
	}

	rows := []r{
		{"1 Town", "fl", 0, 1, 46, [4]int32{0, 1, 2, 3}},
		{"1 Town", "fl", 2, 4, 7, [4]int32{4, -1, -1, -1}},
		{"1 Town", "wl", 0xff, 0xff, 0xff, [4]int32{30, -1, -1, -1}},
		{"1 Cave", "fl", 0, 0, 3, [4]int32{60, -1, -1, -1}},
	}

	b := make([]byte, 4, 4+44*len(rows))
	binary.LittleEndian.PutUint32(b, uint32(len(rows)))

	for _, x := range rows {
		rec := make([]byte, 44)
		copy(rec[0:16], x.lvl)
		copy(rec[16:24], x.tile)
		rec[24], rec[25], rec[26] = x.st, x.a, x.b

		for i, c := range x.cels {
			binary.LittleEndian.PutUint32(rec[28+i*4:], uint32(c))
		}

		b = append(b, rec...)
	}

	return b
}

func mustSynth(t *testing.T) *Table {
	t.Helper()

	tb, err := ParseTxt(strings.NewReader(synthTxt))
	if err != nil {
		t.Fatal(err)
	}

	return tb
}

func TestNames(t *testing.T) {
	cases := []struct {
		name string
		idx  int
	}{{"None", 0}, {"1 Town", 1}, {"1 Cave", 3}, {"2 Town", 12}, {"4 Lava", 28}, {"5 Town", 29}, {"5 Lava", 35}, {"9 Nope", -1}}
	for _, c := range cases {
		if got := LevelIndex(c.name); got != c.idx {
			t.Errorf("LevelIndex(%q)=%d want %d", c.name, got, c.idx)
		}
	}

	for i, n := range []string{"fl", "wl", "wr", "wtlr", "wtll", "wtr", "wbl", "wbr", "wld", "wrd", "wle", "wre", "co"} {
		if TileIndex(n) != i {
			t.Errorf("TileIndex(%s)=%d want %d", n, TileIndex(n), i)
		}
	}
}

func TestLookup(t *testing.T) {
	tb := mustSynth(t)

	cases := []struct {
		name                 string
		lvl, tile, style, sq int
		want                 int // first cel, -1: no row
	}{
		{"floor in range", 1, 0, 0, 1, 0},
		{"range end inclusive", 1, 0, 0, 46, 0},
		{"below range", 1, 0, 0, 0, -1},
		{"above range", 1, 0, 0, 47, -1},
		{"other style", 1, 0, 2, 5, 4},
		{"wall any style any sequence", 1, 1, 17, 200, 30},
		{"wrong orientation", 1, 2, 0, 1, -1},
		{"other level", 3, 0, 0, 2, 60},
		{"level without rows", 2, 0, 0, 0, -1},
	}

	for _, c := range cases {
		r := tb.Lookup(c.lvl, c.tile, c.style, c.sq)
		switch {
		case c.want < 0 && r != nil:
			t.Errorf("%s: unexpected row %+v", c.name, r)
		case c.want >= 0 && (r == nil || r.Cels[0] != c.want):
			t.Errorf("%s: got %+v want first cel %d", c.name, r, c.want)
		}
	}
}

func TestLastBlockWins(t *testing.T) {
	tb, err := ParseTxt(strings.NewReader(blockTxt))
	if err != nil {
		t.Fatal(err)
	}

	if r := tb.Lookup(1, 0, 0, 1); r != nil {
		t.Errorf("first block should be shadowed, got %+v", r)
	}

	if rows := tb.LevelRows(1); len(rows) != 1 || rows[0].Cels[0] != 99 {
		t.Errorf("LevelRows(1)=%+v", rows)
	}
}

func TestCelPickIsModuloCount(t *testing.T) {
	tb := mustSynth(t)

	for pick, want := range map[uint32]int{0: 0, 1: 1, 2: 2, 3: 3, 4: 0, 7: 3} {
		got, ok := tb.Cel(1, 0, 0, 10, pick)
		if !ok || got != want {
			t.Errorf("pick %d: got %d,%v want %d", pick, got, ok, want)
		}
	}

	if got, _ := tb.Cel(1, 0, 2, 5, 12345); got != 4 {
		t.Errorf("single cel row: %d", got)
	}

	if _, ok := tb.Cel(1, 0, 5, 5, 0); ok {
		t.Errorf("no row must give ok=false")
	}
}

func TestBinMatchesTxt(t *testing.T) {
	bin, err := ParseBin(synthBin())
	if err != nil {
		t.Fatal(err)
	}

	txt := mustSynth(t)

	if len(bin.Rows()) != len(txt.Rows()) {
		t.Fatalf("rows %d vs %d", len(bin.Rows()), len(txt.Rows()))
	}

	for i, r := range bin.Rows() {
		o := txt.Rows()[i]
		if r.Level != o.Level || r.Tile != o.Tile || r.Style != o.Style || r.Start != o.Start || r.End != o.End || len(r.Cels) != len(o.Cels) {
			t.Errorf("row %d differs: bin %+v txt %+v", i, r, o)
		}
	}

	if _, err := ParseBin(synthBin()[:20]); err == nil {
		t.Error("truncated bin must fail")
	}
}

func TestProjection(t *testing.T) {
	// hand computed: ((10-4)*8, (10+4)*4)
	if x, y := TileCell(10, 4); x != 48 || y != 56 {
		t.Errorf("TileCell(10,4)=(%d,%d)", x, y)
	}

	if x, y := TileCell(0, 5); x != -40 || y != 20 {
		t.Errorf("TileCell(0,5)=(%d,%d)", x, y)
	}

	if x, y := WorldCell(10.5, 4.5); x != 48 || y != 60 {
		t.Errorf("WorldCell=(%v,%v)", x, y)
	}

	// units: 5 sub-tiles per tile, ((sx-sy)*16, (sx+sy)*8)/10 must equal the tile projection
	sx, sy := 52.5, 22.5 // sub-tiles of tile (10.5, 4.5)
	if ux, uy := (sx-sy)*16/10, (sx+sy)*8/10; ux != 48 || uy != 60 {
		t.Errorf("unit projection (%v,%v)", ux, uy)
	}
}

func TestMovedFar(t *testing.T) {
	cases := []struct {
		dx, dy float64
		want   bool
	}{
		{0, 0, false},
		{7.9, 0, false}, // 79 iso units: (0+2*79)/2 = 79, not > 79
		{8, 0, true},    // 80
		{0, 4, false},   // dy=40: (0+2*40)/2 = 40
		{4, 4, false},   // 40,40: 40+2*40=120/2=60
		{6, 6, true},    // 60,60: 180/2=90
	}
	for _, c := range cases {
		if got := MovedFar(0, 0, c.dx, c.dy); got != c.want {
			t.Errorf("MovedFar(%v,%v)=%v want %v", c.dx, c.dy, got, c.want)
		}
	}
}

func TestLayoutFull800x600(t *testing.T) {
	// hero at tile (10.5,4.5): cell (48,60). hx=48, hy=60;
	// originX = 0x28 + (48-400) = -312, originY = 0xf + (60-300) = -225
	l := ComputeLayout(SizeFull, 800, 600, 48, 60, PanelNone, false)
	if l.OriginX != -312 || l.OriginY != -225 {
		t.Fatalf("origin (%d,%d)", l.OriginX, l.OriginY)
	}

	if x, y := l.HeroScreen(48, 60); x != 360 || y != 285 {
		t.Errorf("hero screen (%d,%d)", x, y)
	}

	// the tile (10,4) cell (48,56) lands 4 px above the hero's position
	if x, y := l.ToScreen(48, 56); x != 360 || y != 281 {
		t.Errorf("tile screen (%d,%d)", x, y)
	}

	// right panel open: the map is shifted left by a quarter of the screen
	r := ComputeLayout(SizeFull, 800, 600, 48, 60, PanelRight, false)
	if x, _ := r.HeroScreen(48, 60); x != 360-200 {
		t.Errorf("panel right hero x %d", x)
	}

	lf := ComputeLayout(SizeFull, 800, 600, 48, 60, PanelLeft, false)
	if x, _ := lf.HeroScreen(48, 60); x != 360+200 {
		t.Errorf("panel left hero x %d", x)
	}
}

func TestLayoutMini800x600(t *testing.T) {
	// scale 20: hx=24 hy=30. Right side: d264=533 d260=78 d210=(266-533)-16=-283 d214=(200-78)-16=106
	// originX = -283+40+(24-400) = -619 ; originY = 106+15+(30-300) = -149
	l := ComputeLayout(SizeMini, 800, 600, 48, 60, PanelNone, false)
	if l.OriginX != -619 || l.OriginY != -149 {
		t.Fatalf("origin (%d,%d)", l.OriginX, l.OriginY)
	}

	if x, y := l.HeroScreen(48, 60); x != 643 || y != 179 {
		t.Errorf("hero (%d,%d)", x, y)
	}

	if l.Clip != (Rect{519, 57, 798, 282}) {
		t.Errorf("clip %+v", l.Clip)
	}

	// the hero must be inside the box
	if x, y := l.HeroScreen(48, 60); !l.Clip.Contains(x, y) {
		t.Error("hero outside the mini box")
	}

	// a cell is half as far from the hero as in the full map
	if x, _ := l.ToScreen(48+16, 60); x != 643+8 {
		t.Errorf("scaled x %d", x)
	}
}

func TestModelDedup(t *testing.T) {
	m := NewModel()

	steps := []struct {
		cel  int
		want bool
	}{
		{0, true},    // first cell
		{0, false},   // same cel
		{1, false},   // same group 0
		{6, true},    // group 1 differs from group 0
		{100, false}, // ungrouped cel at an occupied position
	}

	for _, s := range steps {
		if got := m.Add(LayerFloor, 8, 4, s.cel); got != s.want {
			t.Errorf("Add cel %d = %v want %v", s.cel, got, s.want)
		}
	}

	if !m.Add(LayerWall, 8, 4, 100) {
		t.Error("another layer is independent")
	}

	if m.Count() != 3 || m.CountLayer(LayerFloor) != 2 || m.CountLayer(LayerWall) != 1 {
		t.Errorf("counts %d %d %d", m.Count(), m.CountLayer(LayerFloor), m.CountLayer(LayerWall))
	}
}

// grid is a w x h map of single floors, plus a wall on tile (1,1).
type grid struct{ w, h int }

func (g grid) Size() (int, int) { return g.w, g.h }

func (g grid) Tiles(tx, ty int) []TileRef {
	refs := []TileRef{{LayerFloor, 0, 0, 1}} // floor style 0 sequence 1
	if tx == 1 && ty == 1 {
		refs = append(refs, TileRef{LayerWall, 1, 3, 3})
	}

	return refs
}

func TestRevealSynthetic(t *testing.T) {
	tb := mustSynth(t)
	m := NewModel()
	src := grid{3, 3}

	// hero at the centre with a big area: all 9 floors and the wall (cel 30)
	if n := m.RevealTiles(tb, src, 1, 7, 1.5, 1.5, Area{200, 200}); n != 10 {
		t.Fatalf("revealed %d, want 9 floors + 1 wall", n)
	}

	if m.CountLayer(LayerFloor) != 9 || m.CountLayer(LayerWall) != 1 {
		t.Errorf("layers %d/%d", m.CountLayer(LayerFloor), m.CountLayer(LayerWall))
	}

	// walking again over the same tiles reveals nothing new (each tile is processed once)
	if n := m.RevealTiles(tb, src, 1, 7, 1.5, 1.5, Area{200, 200}); n != 0 {
		t.Errorf("second reveal added %d", n)
	}

	cells := m.Cells()
	// floors first, sorted by y then x: (0,0)->(0,0); y=4: (0,1)->(-8,4) and (1,0)->(8,4); ...
	want := [][2]int{{0, 0}, {-8, 4}, {8, 4}}
	for i, w := range want {
		if cells[i].Kind != LayerFloor || cells[i].X != w[0] || cells[i].Y != w[1] {
			t.Errorf("cell %d = %+v want pos %v", i, cells[i], w)
		}
	}

	last := cells[len(cells)-1]
	// the wall on tile (1,1): cell (0, 8), cel 30
	if last.Kind != LayerWall || last.X != 0 || last.Y != 8 || last.Cel != 30 {
		t.Errorf("wall cell %+v", last)
	}

	// each floor cel is one of the four of the row
	for _, c := range cells {
		if c.Kind == LayerFloor && (c.Cel < 0 || c.Cel > 3) {
			t.Errorf("floor cel %d", c.Cel)
		}
	}
}

func TestRevealRadius(t *testing.T) {
	tb := mustSynth(t)
	src := grid{40, 40}
	m := NewModel()

	// hero at tile (20.5,20.5), area +-24 cell px in x and +-12 in y:
	// |tx-ty|*8 <= 24 and |(tx+ty-41)|*4 <= 12 -> tx-ty in [-3,3] and tx+ty in [38,44]
	n := m.RevealTiles(tb, src, 1, 1, 20.5, 20.5, Area{24, 12})

	want := 0

	for ty := 0; ty < 40; ty++ {
		for tx := 0; tx < 40; tx++ {
			cx, cy := TileCell(tx, ty)
			if abs(float64(cx)-0) <= 24 && abs(float64(cy)-164) <= 12 {
				want++
			}
		}
	}

	// the hero cell is (0, 164); floors on style 0 seq 1 all have a row
	if n < want {
		t.Errorf("revealed %d, expected at least the %d floors in the area", n, want)
	}

	if n > want+1 { // + the (1,1)-wall only if it were inside, it is not
		t.Errorf("revealed %d, expected %d", n, want)
	}

	// a tile far away stays hidden
	for _, c := range m.Cells() {
		if abs(float64(c.X)) > 24 || abs(float64(c.Y)-164) > 12 {
			t.Errorf("cell %+v outside the area", c)
		}
	}
}

func TestAddObject(t *testing.T) {
	m := NewModel()
	// object at tile (10.5, 4.5): cell (48,60) -> (+1, -3) = (49, 57)
	if !m.AddObject(5, 10.5, 4.5) {
		t.Fatal("object not added")
	}

	c := m.Cells()[0]
	if c.X != 49 || c.Y != 57 || c.Kind != LayerObject || c.Cel != 5 {
		t.Errorf("object cell %+v", c)
	}

	if m.AddObject(0, 1, 1) {
		t.Error("cel 0 means no automap graphic")
	}
}

// Real tables: D2_TABLES=$HOME/git/d2-tables
func TestRealTables(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	f, err := os.Open(filepath.Join(root, "drlg", "d2exp", "AutoMap.txt"))
	if err != nil {
		t.Skip("no AutoMap.txt: ", err)
	}
	defer f.Close()

	txt, err := ParseTxt(f)
	if err != nil {
		t.Fatal(err)
	}

	if len(txt.Rows()) != 3286 {
		t.Errorf("rows %d, want 3286 (the compiled automap.bin has 0xcd6)", len(txt.Rows()))
	}

	// Rogue Encampment (Act 1 town): row "1 Town fl 2 8 11 -> River M A/B/C = 6 7 8"
	for pick, want := range map[uint32]int{0: 6, 1: 7, 2: 8, 3: 6} {
		if got, ok := txt.Cel(1, 0, 2, 9, pick); !ok || got != want {
			t.Errorf("town river M pick %d: %d,%v want %d", pick, got, ok, want)
		}
	}

	if got, ok := txt.Cel(1, 0, 2, 5, 99); !ok || got != 4 {
		t.Errorf("town river T: %d,%v", got, ok)
	}

	// floor style 0 sequences 1..46 are the path tiles: cels 0-3
	if r := txt.Lookup(1, 0, 0, 20); r == nil || len(r.Cels) != 4 || r.Cels[0] != 0 {
		t.Errorf("town path row %+v", r)
	}

	// each level of the table forms a single block, so "last block wins" does not drop rows
	seen := map[int]bool{}

	rows := txt.Rows()
	for i := range rows {
		if i > 0 && rows[i].Level == rows[i-1].Level {
			continue
		}

		if seen[rows[i].Level] {
			t.Logf("level %s occurs in several blocks (last wins)", LevelName(rows[i].Level))
		}

		seen[rows[i].Level] = true
	}

	// all MaxiMap cels are < 1499 (the DC6 has 1499 frames)
	for _, r := range rows {
		for _, c := range r.Cels {
			if c < 0 || c >= 1499 {
				t.Fatalf("cel %d out of range in %+v", c, r)
			}
		}
	}

	// the compiled automap.bin (extracted from d2exp.mpq) must give identical rows
	b, err := os.ReadFile(filepath.Join(root, "drlg", "bin", "d2exp", "automap.bin"))
	if err != nil {
		t.Log("no automap.bin extracted, skipping the txt/bin comparison")
		return
	}

	bin, err := ParseBin(b)
	if err != nil {
		t.Fatal(err)
	}

	if len(bin.Rows()) != len(txt.Rows()) {
		t.Fatalf("bin rows %d txt rows %d", len(bin.Rows()), len(txt.Rows()))
	}

	for i, r := range bin.Rows() {
		o := txt.Rows()[i]
		if r.Level != o.Level || r.Tile != o.Tile || r.Style != o.Style || r.Start != o.Start || r.End != o.End {
			t.Fatalf("row %d: bin %+v txt %+v", i, r, o)
		}

		if len(r.Cels) != len(o.Cels) {
			t.Fatalf("row %d cels: bin %v txt %v", i, r.Cels, o.Cels)
		}

		for k := range r.Cels {
			if r.Cels[k] != o.Cels[k] {
				t.Fatalf("row %d cel %d: bin %v txt %v", i, k, r.Cels, o.Cels)
			}
		}
	}
}
