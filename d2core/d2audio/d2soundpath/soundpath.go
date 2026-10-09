package d2soundpath

import "strings"

// Roots are the archive folders searched, in order, for a relative sound name.
var Roots = []string{"data/global/sfx/", "data/local/sfx/", "data/global/music/"}

// Normalize converts a sound name to forward slashes without a leading slash.
func Normalize(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")

	return strings.TrimLeft(name, "/")
}

// Candidates lists the archive paths to try for name, most likely first. A name that already
// starts with "data/" is a full archive path and is used as-is.
func Candidates(name string) []string {
	n := Normalize(name)
	if n == "" {
		return nil
	}

	if strings.HasPrefix(strings.ToLower(n), "data/") {
		return []string{n}
	}

	out := make([]string, 0, len(Roots))
	for _, r := range Roots {
		out = append(out, r+n)
	}

	return out
}

// Resolve returns the first candidate path for which exists reports true.
func Resolve(name string, exists func(string) bool) (string, bool) {
	for _, c := range Candidates(name) {
		if exists(c) {
			return c, true
		}
	}

	return "", false
}
