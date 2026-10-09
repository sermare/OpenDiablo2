package d2hero

import (
	"os"
	"sync"
)

//nolint:gochecknoglobals // process-wide import folders, set once at startup
var (
	importDirsMu sync.RWMutex
	importDirs   []string
)

// SetImportDirs sets the folders of real Diablo II .d2s characters that are
// imported (read only) into the character list, in addition to OD2_D2S_DIR.
func SetImportDirs(dirs []string) {
	importDirsMu.Lock()
	defer importDirsMu.Unlock()

	importDirs = append([]string(nil), dirs...)
}

// ImportDirs returns OD2_D2S_DIR (if set) followed by the folders given to
// SetImportDirs, without duplicates.
func ImportDirs() []string {
	importDirsMu.RLock()
	defer importDirsMu.RUnlock()

	seen := map[string]bool{}
	out := make([]string, 0, len(importDirs)+1)

	for _, d := range append([]string{os.Getenv("OD2_D2S_DIR")}, importDirs...) {
		if d != "" && !seen[d] {
			seen[d] = true

			out = append(out, d)
		}
	}

	return out
}
