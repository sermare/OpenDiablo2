package d2hero

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A reader that races a writer of the same file (a host and a joiner loading a
// hero while another process saves it) must see the old or the new content in
// full, never an empty or half-written file.
func TestWriteFileAtomicNeverShowsPartialFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "0.od2")
	small, big := []byte("{}"), []byte(strings.Repeat("x", 1<<20))

	if err := os.WriteFile(path, small, 0o600); err != nil {
		t.Fatal(err)
	}

	done := make(chan struct{})
	werr := make(chan error, 1)

	go func() {
		defer close(done)

		for i := 0; i < 200; i++ {
			data := small
			if i%2 == 0 {
				data = big
			}

			if err := writeFileAtomic(path, data); err != nil {
				werr <- err
				return
			}
		}
	}()

	for running := true; running; {
		select {
		case <-done:
			running = false
		default:
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the file vanished while it was replaced: %v", err)
		}

		if len(got) != len(small) && len(got) != len(big) {
			t.Fatalf("read a partial file: %d bytes", len(got))
		}
	}

	select {
	case err := <-werr:
		t.Fatal(err)
	default:
	}

	if entries, _ := os.ReadDir(filepath.Dir(path)); len(entries) != 1 {
		t.Fatalf("temporary files left behind: %d entries", len(entries))
	}
}
