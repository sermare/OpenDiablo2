package d2mpq

import (
	"bytes"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/JoshVarga/blast"
)

// blastDecompress is the reference decoder explode replaced.
func blastDecompress(data []byte) (out []byte, err error) {
	defer func() { // the reference panics on some streams the blast writer produces
		if r := recover(); r != nil {
			out, err = nil, errExplodeCode
		}
	}()

	r, err := blast.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	return io.ReadAll(r)
}

// blastCompress compresses with the library's writer, which panics on some inputs.
func blastCompress(data []byte, typ, dict uint) (cmp *bytes.Buffer, ok bool) {
	defer func() {
		if recover() != nil {
			cmp, ok = nil, false
		}
	}()

	cmp = new(bytes.Buffer)
	w := blast.NewWriter(cmp, typ, dict)
	_, _ = w.Write(data)
	_ = w.Close()

	return cmp, true
}

func TestExplodeKnownVector(t *testing.T) {
	got, err := explode([]byte{0x00, 0x04, 0x82, 0x24, 0x25, 0x8f, 0x80, 0x7f}, 0)
	if err != nil || string(got) != "AIAIAIAIAIAIA" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestExplodeMatchesBlast(t *testing.T) {
	rnd := rand.New(rand.NewSource(7))

	gen := map[string]func(n int) []byte{
		"random": func(n int) []byte { b := make([]byte, n); rnd.Read(b); return b },
		"few symbols": func(n int) []byte {
			b := make([]byte, n)
			for i := range b {
				b[i] = byte(rnd.Intn(5))
			}

			return b
		},
		"runs and repeats": func(n int) []byte {
			var b []byte
			for len(b) < n {
				if len(b) > 10 && rnd.Intn(3) == 0 {
					s := rnd.Intn(len(b) - 5)
					b = append(b, b[s:s+2+rnd.Intn(300)%(len(b)-s-1)]...)
				} else {
					b = append(b, bytes.Repeat([]byte{byte(rnd.Intn(256))}, 1+rnd.Intn(40))...)
				}
			}

			return b[:n]
		},
	}

	// a fixed order: the generators share one random stream, and with a random map order some streams make
	// the third party blast reader loop forever (a flaky hang of the whole test binary)
	names := make([]string, 0, len(gen))
	for name := range gen {
		names = append(names, name)
	}

	sort.Strings(names)

	for _, name := range names {
		g := gen[name]

		for _, typ := range []uint{blast.Binary, blast.ASCII} {
			for _, dict := range []uint{blast.DictionarySize1024, blast.DictionarySize2048, blast.DictionarySize4096} {
				for _, n := range []int{1, 2, 3, 100, 4095, 4096, 4097, 20000} {
					data := g(n)

					cmp, ok := blastCompress(data, typ, dict)
					if !ok {
						continue
					}

					// the blast writer emits broken streams for some larger incompressible inputs (the old
					// reader fails on them too): only streams the old reader decodes to the input are used
					if ref, err := blastDecompress(cmp.Bytes()); err != nil || !bytes.Equal(ref, data) {
						continue
					}

					for _, hint := range []int{0, n, 10} {
						got, err := explode(cmp.Bytes(), hint)
						if err != nil {
							t.Fatalf("%s/%d/%d/%d: %v", name, typ, dict, n, err)
						}

						if !bytes.Equal(got, data) {
							t.Fatalf("%s/%d/%d/%d: output differs", name, typ, dict, n)
						}
					}
				}
			}
		}
	}
}

func TestExplodeRejectsBadInput(t *testing.T) {
	tests := []struct {
		name string
		in   []byte
	}{
		{"empty", nil},
		{"bad literal flag", []byte{2, 4, 0, 0}},
		{"bad dictionary", []byte{0, 7, 0, 0}},
		{"truncated", []byte{0x00, 0x04, 0x82, 0x24}},
		{"no end code", bytes.Repeat([]byte{0}, 40)[:40]},
	}

	for _, tt := range tests {
		if tt.name == "no end code" {
			tt.in = append([]byte{0, 4}, tt.in...)
		}

		if _, err := explode(tt.in, 0); err == nil {
			t.Errorf("%s: no error", tt.name)
		}
	}

	// a distance that points before the start of the output
	if _, err := explode([]byte{0x00, 0x04, 0x03, 0x00, 0x00, 0x00}, 0); err == nil {
		t.Error("distance before the start: no error")
	}
}

// TestExplodeRealArchive decodes every file of a game archive with both decoders. It needs
// D2_GAME_DIR (the install folder) and reads archives only.
func TestExplodeRealArchive(t *testing.T) {
	dir := os.Getenv("D2_GAME_DIR")
	if dir == "" {
		t.Skip("D2_GAME_DIR not set")
	}

	m, err := FromFile(filepath.Join(dir, "d2data.mpq"))
	if err != nil {
		t.Skip(err)
	}

	defer m.Close()

	checked := 0

	for h := range m.hashes {
		_ = h
		break
	}

	for _, b := range m.blocks {
		if b.UncompressedFileSize == 0 || b.UncompressedFileSize > 1<<20 || b.HasFlag(FileEncrypted) || b.HasFlag(FileSingleUnit) {
			continue
		}

		if !b.HasFlag(FileCompress) && !b.HasFlag(FileImplode) {
			continue
		}

		st := &Stream{MPQ: m, Block: b, Index: 0xFFFFFFFF}
		st.Size = 0x200 << m.header.BlockSize

		if st.loadBlockOffsets() != nil {
			continue
		}

		for i := 0; i+1 < len(st.Positions); i++ {
			raw := make([]byte, st.Positions[i+1]-st.Positions[i])
			if _, err := m.file.ReadAt(raw, int64(b.FilePosition+st.Positions[i])); err != nil {
				break
			}

			want := uint32(0)
			if want = b.UncompressedFileSize - uint32(i)*st.Size; want > st.Size {
				want = st.Size
			}

			if uint32(len(raw)) == want || len(raw) < 2 {
				continue
			}

			body, isMulti := raw, false
			if b.HasFlag(FileCompress) {
				if raw[0] != 8 { // only the PKWare method
					continue
				}

				body, isMulti = raw[1:], true
			}

			_ = isMulti

			ref, err1 := blastDecompress(body)
			got, err2 := explode(body, int(want))

			if (err1 == nil) != (err2 == nil) || !bytes.Equal(ref, got) {
				t.Fatalf("sector %d of block %+v differs (ref err %v, got err %v)", i, b.FilePosition, err1, err2)
			}

			checked++
		}

		if checked > 3000 {
			break
		}
	}

	t.Logf("%d imploded sectors identical", checked)

	if checked == 0 {
		t.Log("no imploded sectors found in the archive")
	}
}

// BenchmarkExplode decodes imploded sectors of the real archive (needs D2_GAME_DIR) with the new and
// the old decoder.
func BenchmarkExplode(b *testing.B) {
	dir := os.Getenv("D2_GAME_DIR")
	if dir == "" {
		b.Skip("D2_GAME_DIR not set")
	}

	m, err := FromFile(filepath.Join(dir, "d2data.mpq"))
	if err != nil {
		b.Skip(err)
	}

	defer m.Close()

	var sectors [][]byte

	total := 0

	for _, blk := range m.blocks {
		if blk.UncompressedFileSize < 8192 || blk.UncompressedFileSize > 1<<20 || blk.HasFlag(FileEncrypted) ||
			blk.HasFlag(FileSingleUnit) || !blk.HasFlag(FileCompress) {
			continue
		}

		st := &Stream{MPQ: m, Block: blk}
		st.Size = 0x200 << m.header.BlockSize

		if st.loadBlockOffsets() != nil || len(st.Positions) < 3 {
			continue
		}

		raw := make([]byte, st.Positions[1]-st.Positions[0])
		if _, err := m.file.ReadAt(raw, int64(blk.FilePosition+st.Positions[0])); err != nil || raw[0] != 8 {
			continue
		}

		sectors = append(sectors, raw[1:])
		total += int(st.Size)

		if len(sectors) == 400 {
			break
		}
	}

	b.Run("explode", func(b *testing.B) {
		b.SetBytes(int64(total))

		for i := 0; i < b.N; i++ {
			for _, sec := range sectors {
				if _, err := explode(sec, int(0x1000)); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
	b.Run("blast", func(b *testing.B) {
		b.SetBytes(int64(total))

		for i := 0; i < b.N; i++ {
			for _, sec := range sectors {
				if _, err := blastDecompress(sec); err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}
