package d2realmclient

import (
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2networking/d2realm"
)

// CharName turns an engine hero name into a name the realm accepts for a
// character (2 to 15 letters, single - or _ inside): other characters are
// dropped, a name that is too short becomes "Hero" plus what is left.
func CharName(name string) string {
	var b strings.Builder

	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case (r == '-' || r == '_') && b.Len() > 0:
			b.WriteRune(r)
		}
	}

	out := strings.TrimRight(b.String(), "-_")
	if len(out) > 15 {
		out = out[:15]
	}

	if len(out) < 2 {
		out = "Hero" + out
	}

	return out
}

// AccountName is the realm account of a hero: its character name (accounts
// are unique online, so two copies of one hero cannot join one realm).
func AccountName(name string) string {
	a := CharName(name)
	if !d2realm.ValidAccount(a) {
		return "Hero"
	}

	return a
}
