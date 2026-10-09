package d2drlg

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadInline(t *testing.T) {
	tb, err := Load(Raw{
		Levels:   []byte("Name\tId\tSizeX\tSizeY\tSizeX(N)\tSizeY(N)\tSizeX(H)\tSizeY(H)\tOffsetX\tOffsetY\tDepend\tDrlgType\tLevelType\tVis0\nNull\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\t0\nCave\t8\t200\t200\t200\t200\t200\t200\t1500\t1000\t0\t1\t3\t9\n"),
		LvlMaze:  []byte("Name\tLevel\tRooms\tRooms(N)\tRooms(H)\tSizeX\tSizeY\tMerge\nA\t0\t1\t1\t1\t24\t24\t0\nB\t8\t1\t2\t3\t24\t24\t500\n"),
		LvlPrest: []byte("Name\tDef\tLevelId\tSizeX\tSizeY\tFiles\tFile1\tFile2\nsep\t\t\t\t\t\t\t\nCave N\t60\t0\t24\t24\t2\ta.ds1\tb.ds1\n"),
	})
	if err != nil {
		t.Fatal(err)
	}

	l, ok := tb.Level(8)
	if !ok || l.OffsetX != 1500 || l.DrlgType != 1 || l.LevelType != 3 || l.Vis[0] != 9 {
		t.Fatalf("level: %+v", l)
	}

	if _, ok := tb.Level(0); ok {
		t.Fatal("Null level must be skipped")
	}

	m, ok := tb.Maze(8)
	if !ok || m.Rooms != [3]int{1, 2, 3} || m.Merge != 500 {
		t.Fatalf("maze: %+v", m)
	}

	p, ok := tb.PrestByDef(60)
	if !ok || p.Files != 2 || p.File[1] != "b.ds1" || tb.PrestN != 1 {
		t.Fatalf("prest: %+v n=%d", p, tb.PrestN)
	}
}

func TestPrestBinBadSize(t *testing.T) {
	if _, err := ParseLvlPrestBin([]byte{1, 0, 0, 0, 9}); err == nil {
		t.Fatal("expected size error")
	}
}

// TestRealTablesBinMatchesTxt checks that the compiled lvlprest.bin agrees with
// the extracted txt (Def, size, Files, file names) for every non-separator row.
func TestRealTablesBinMatchesTxt(t *testing.T) {
	root := os.Getenv("D2_TABLES")
	if root == "" {
		t.Skip("D2_TABLES not set")
	}

	rd := func(p string) []byte {
		b, err := os.ReadFile(filepath.Join(root, "drlg", p))
		if err != nil {
			t.Skip(err)
		}

		return b
	}

	txt, err := parsePrestTxt(rd("patch_d2/LvlPrest.txt"))
	if err != nil {
		t.Fatal(err)
	}

	bin, err := ParseLvlPrestBin(rd("bin/patch_d2/lvlprest.bin"))
	if err != nil {
		t.Fatal(err)
	}

	if len(txt) != len(bin) {
		t.Fatalf("txt %d records, bin %d", len(txt), len(bin))
	}

	for i := range bin {
		a, b := txt[i], bin[i]
		a.Name = ""

		if a != b {
			t.Fatalf("record %d differs:\n txt %+v\n bin %+v", i, a, b)
		}
	}

	tb, err := Load(Raw{Levels: rd("patch_d2/Levels.txt"), LvlMaze: rd("patch_d2/LvlMaze.txt"), LvlPrest: rd("patch_d2/LvlPrest.txt"),
		LvlPrestBin: rd("bin/patch_d2/lvlprest.bin"), LvlTypes: rd("patch_d2/LvlTypes.txt"), LvlSub: rd("patch_d2/LvlSub.txt")})
	if err != nil {
		t.Fatal(err)
	}

	// Independent facts read from the tables themselves.
	p, _ := tb.PrestByDef(53)
	if p.Name != "Act 1 - Cave W" || p.Files != 2 || p.SizeX != 24 {
		t.Errorf("cave W: %+v", p)
	}

	m, _ := tb.Maze(18)
	if m.Rooms != [3]int{12, 24, 36} || m.SizeX != 8 {
		t.Errorf("crypt 1 maze: %+v", m)
	}

	town, _ := tb.PrestByDef(1)
	if town.Files != 0 || town.File[0] != "Act1/Town/TownN1.ds1" {
		t.Errorf("town: %+v", town) // Files==0: the variant is forced by the world layout, not rolled
	}

	if len(tb.SubRows(0)) == 0 {
		t.Error("no LvlSub rows of type 0")
	}
}
