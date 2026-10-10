package d2drlg

import (
	"encoding/binary"
	"testing"
)

// synthPrestBin builds a lvlprest.bin with n zeroed records.
func synthPrestBin(n int) []byte {
	b := make([]byte, 4+n*binRecordSize)
	binary.LittleEndian.PutUint32(b, uint32(n))

	return b
}

// FuzzParseLvlPrestBin feeds arbitrary bytes to the compiled lvlprest.bin decoder.
func FuzzParseLvlPrestBin(f *testing.F) {
	f.Add(synthPrestBin(0))
	f.Add(synthPrestBin(2))
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = ParseLvlPrestBin(data)
	})
}

// FuzzLoadTables feeds arbitrary table text and bin data to the DRLG table loader.
func FuzzLoadTables(f *testing.F) {
	levels := "Name\tId\tAct\tSizeX\tSizeY\tVis0\tWarp0\nNull\t0\t0\t0\t0\t0\t0\nRogue Encampment\t1\t0\t10\t10\t2\t1\n"
	maze := "Name\tLevel\tRooms\tSizeX\tSizeY\nCatacombs\t34\t5\t4\t4\n"
	prest := "Name\tDef\tLevelId\tSizeX\tSizeY\tFiles\tFile1\n" + "Town\t1\t1\t8\t8\t1\tdata\\a.ds1\n"

	f.Add([]byte(levels), []byte(maze), []byte(prest), synthPrestBin(1))
	f.Add([]byte(""), []byte("\t\t\n"), []byte("Def\n1\n"), []byte{})

	f.Fuzz(func(t *testing.T, lv, mz, pr, bin []byte) {
		_, _ = Load(Raw{Levels: lv, LvlMaze: mz, LvlPrest: pr, LvlPrestBin: bin})
	})
}
