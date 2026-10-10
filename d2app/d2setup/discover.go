package d2setup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// CoreMPQs are the archives the base game needs.
//
//nolint:gochecknoglobals // constant table
var CoreMPQs = []string{
	"patch_d2.mpq", "d2data.mpq", "d2char.mpq", "d2music.mpq",
	"d2sfx.mpq", "d2speech.mpq", "d2video.mpq",
}

// ExpansionMPQs are the Lord of Destruction archives.
//
//nolint:gochecknoglobals // constant table
var ExpansionMPQs = []string{"d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq"}

// LoadOrder is the order the engine mounts archives in (later entries lose to
// earlier ones, so the patch comes first).
//
//nolint:gochecknoglobals // constant table
var LoadOrder = []string{
	"patch_d2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq",
	"d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2video.mpq", "d2speech.mpq",
}

// Validation is the result of inspecting a folder for game files.
type Validation struct {
	Dir              string
	MissingCore      []string
	MissingExpansion []string
	// LoadOrder lists the archives that exist, with the file name spelled as
	// on disk, in engine mount order.
	LoadOrder []string
}

// OK reports whether the base game files are all present.
func (v Validation) OK() bool { return len(v.MissingCore) == 0 }

// Validate checks dir for the Diablo II archives, ignoring file name case.
func Validate(dir string) Validation {
	v := Validation{Dir: dir}
	actual := map[string]string{}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if !e.IsDir() {
			actual[strings.ToLower(e.Name())] = e.Name()
		}
	}

	for _, want := range LoadOrder {
		if name, ok := actual[want]; ok {
			v.LoadOrder = append(v.LoadOrder, name)
		}
	}

	for _, want := range CoreMPQs {
		if _, ok := actual[want]; !ok {
			v.MissingCore = append(v.MissingCore, want)
		}
	}

	for _, want := range ExpansionMPQs {
		if _, ok := actual[want]; !ok {
			v.MissingExpansion = append(v.MissingExpansion, want)
		}
	}

	return v
}

// Resolve validates dir and, when the archives are not directly inside it, the
// usual sub folder "Diablo II" (people often pick the parent folder).
func Resolve(dir string) Validation {
	v := Validate(dir)
	if v.OK() {
		return v
	}

	for _, sub := range []string{"Diablo II", "Diablo 2", "diablo ii"} {
		if sv := Validate(filepath.Join(dir, sub)); sv.OK() {
			return sv
		}
	}

	return v
}

// Complete reports whether every archive in order exists in dir.
func Complete(dir string, order []string) bool {
	if dir == "" || len(order) == 0 {
		return false
	}

	for _, name := range order {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			return false
		}
	}

	return true
}

func glob(pattern string) []string {
	m, _ := filepath.Glob(pattern)
	sort.Strings(m)

	return m
}

// gameDirCandidates lists (never ~/Documents, ~/Desktop or ~/Downloads: macOS asks the user for permission when an app reads them, which would stall a Finder launch) the places a Diablo II install is usually found on a Mac.
func gameDirCandidates(home string) []string {
	out := []string{
		"/Applications/Diablo II",
		"/Applications/Diablo II/Diablo II",
		"/Applications/Diablo 2",
		"/Users/Shared/Diablo II",
		filepath.Join(home, "Applications", "Diablo II"),
		filepath.Join(home, "Library", "Application Support", "Diablo II"),
		filepath.Join(home, "Library", "Application Support", "Blizzard", "Diablo II"),
		filepath.Join(home, "Games", "Diablo II"),
		// Steam and Battle.net (Mac) installs, when they hold the classic files
		filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "common", "Diablo II"),
		filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "common", "Diablo II Classic"),
		filepath.Join("/Applications", "Battle.net", "Diablo II"),
	}

	for _, drive := range []string{
		filepath.Join(home, ".wine*", "drive_c"),
		filepath.Join(home, "Library", "Application Support", "CrossOver", "Bottles", "*", "drive_c"),
		filepath.Join(home, "Library", "Application Support", "Battle.net", "drive_c"),
		filepath.Join(home, "Library", "Containers", "com.isaacmarovitz.Whisky", "Bottles", "*", "drive_c"),
		filepath.Join(home, "Library", "Application Support", "Steam", "steamapps", "compatdata", "*", "pfx", "drive_c"),
	} {
		for _, pf := range []string{"Program Files (x86)", "Program Files"} {
			out = append(out, glob(filepath.Join(drive, pf, "Diablo II"))...)
		}
	}

	return out
}

// FindInstalls returns the folders that hold a complete set of base game
// archives, the configured one (extra) first. Duplicates are removed.
func FindInstalls(home string, extra ...string) []Validation {
	seen := map[string]bool{}

	var found []Validation

	cands := make([]string, 0, len(extra)+10)

	for _, e := range extra {
		if e != "" {
			cands = append(cands, e)
		}
	}

	cands = append(cands, gameDirCandidates(home)...)

	for _, c := range cands {
		c = filepath.Clean(c)
		if seen[c] {
			continue
		}

		seen[c] = true

		if v := Validate(c); v.OK() {
			found = append(found, v)
		}
	}

	return found
}

// SaveDirCandidates lists the folders that may hold real .d2s characters.
// gameDir is the configured game folder (1.14 may also keep a Save folder there).
func SaveDirCandidates(home, gameDir string) []string {
	out := []string{}

	for _, pat := range []string{
		filepath.Join(home, "Library", "Application Support", "Diablo II*"),
		filepath.Join(home, "Library", "Application Support", "Diablo II*", "Save"),
		filepath.Join(home, "Library", "Application Support", "Blizzard", "Diablo II*"),
		filepath.Join(home, "Library", "Application Support", "Blizzard", "Diablo II*", "Save"),
		filepath.Join(home, ".wine*", "drive_c", "users", "*", "Saved Games", "Diablo II*"),
		filepath.Join(home, ".wine*", "drive_c", "Program Files*", "Diablo II", "Save"),
		filepath.Join(home, "Library", "Application Support", "CrossOver", "Bottles", "*",
			"drive_c", "users", "*", "Saved Games", "Diablo II*"),
	} {
		out = append(out, glob(pat)...)
	}

	if gameDir != "" {
		out = append(out, filepath.Join(gameDir, "Save"), filepath.Join(gameDir, "save"))
	}

	return out
}

// FindSaveDirs returns the candidate folders that contain at least one .d2s
// file, deduplicated, in discovery order. extra is checked first (explicit config).
func FindSaveDirs(home, gameDir string, extra ...string) []string {
	seen := map[string]bool{}

	var out []string

	for _, d := range append(append([]string{}, extra...), SaveDirCandidates(home, gameDir)...) {
		if d == "" {
			continue
		}

		d = filepath.Clean(d)
		if seen[d] || !hasD2S(d) {
			continue
		}

		seen[d] = true

		out = append(out, d)
	}

	return out
}

func hasD2S(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}

	for _, e := range entries {
		if !e.IsDir() && strings.EqualFold(filepath.Ext(e.Name()), ".d2s") {
			return true
		}
	}

	return false
}
