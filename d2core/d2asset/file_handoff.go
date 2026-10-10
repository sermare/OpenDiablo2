package d2asset

import "sync"

// handoffBudget caps the bytes kept for the second reader of a file (see LoadFileHandoff).
const handoffBudget = 24 << 20

// fileHandoff keeps the bytes of files that two independent consumers read once each.
type fileHandoff struct {
	mu    sync.Mutex
	files map[string][]byte
	bytes int
}

// LoadFileHandoff is LoadFile for files that exactly two consumers read, each only once and each
// keeping its own parsed copy (a level's DT1 tile files are read by the map engine and by the DRLG
// outdoor generator). The first call reads and decompresses the file and remembers the bytes; the
// second call returns them and forgets them, so a tile file is decompressed once instead of
// twice and the raw bytes are not kept afterwards. A file that is only ever read once stays in the
// (bounded) handoff table, so the table is capped. The returned slice must not be modified.
func (am *AssetManager) LoadFileHandoff(filePath string) ([]byte, error) {
	h := &am.handoff

	h.mu.Lock()

	if b, ok := h.files[filePath]; ok {
		delete(h.files, filePath)
		h.bytes -= len(b)
		h.mu.Unlock()

		return b, nil
	}

	h.mu.Unlock()

	b, err := am.LoadFile(filePath)
	if err != nil {
		return nil, err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if h.bytes+len(b) <= handoffBudget {
		if h.files == nil {
			h.files = map[string][]byte{}
		}

		if old, dup := h.files[filePath]; dup {
			h.bytes -= len(old)
		}

		h.files[filePath] = b
		h.bytes += len(b)
	}

	return b, nil
}
