package d2cube

import (
	"errors"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2core/d2item/d2drop"
)

// Runeword is a row of Runes.txt (the runeword table).
type Runeword struct {
	ID       string // "Runeword1"
	Name     string
	Complete bool // only complete runewords work in the game
	Ladder   bool // the "server" column: only on ladder / realm games
	Include  []string
	Exclude  []string
	Runes    []string // rune codes in socket order
}

// Runewords is the parsed Runes.txt.
type Runewords struct {
	List []*Runeword
}

// ParseRunewords reads Runes.txt.
func ParseRunewords(raw []byte) *Runewords {
	t := parseTable(raw)
	out := &Runewords{}

	for _, r := range t.rows {
		name := t.s(r, "rune name")
		if name == "" {
			continue
		}

		rw := &Runeword{ID: t.s(r, "name"), Name: name, Complete: t.b(r, "complete"), Ladder: t.b(r, "server")}

		for k := 1; k <= 6; k++ {
			if v := t.s(r, "itype"+string(rune('0'+k))); v != "" {
				rw.Include = append(rw.Include, v)
			}
		}

		for k := 1; k <= 3; k++ {
			if v := t.s(r, "etype"+string(rune('0'+k))); v != "" {
				rw.Exclude = append(rw.Exclude, v)
			}
		}

		for k := 1; k <= 6; k++ {
			if v := t.s(r, "rune"+string(rune('0'+k))); v != "" {
				rw.Runes = append(rw.Runes, v)
			}
		}

		out.List = append(out.List, rw)
	}

	return out
}

// Errors of socketing.
var (
	ErrNotSocketable = errors.New("that item cannot be put into a socket")
	ErrNoFreeSocket  = errors.New("the item has no free socket")
	ErrNoSockets     = errors.New("the item has no sockets")
)

// CanSocket checks that filler (a gem, rune or jewel code) can go into one of
// the item's free sockets.
func (cat *Catalog) CanSocket(it *Item, filler string) error {
	fb, ok := cat.bases[filler]
	if !ok || !fb.HasType("sock") {
		return ErrNotSocketable
	}

	if it.Sockets == 0 {
		return ErrNoSockets
	}

	if len(it.Socketed) >= it.Sockets {
		return ErrNoFreeSocket
	}

	return nil
}

// Socket puts a gem, rune or jewel into the item's next free socket. When this
// fills the last socket and the runes spell a runeword that fits the item, the
// item becomes that runeword (it.Runeword is set). It returns the runeword or
// nil.
func (cat *Catalog) Socket(it *Item, filler string, rw *Runewords, ladder bool) (*Runeword, error) {
	if err := cat.CanSocket(it, filler); err != nil {
		return nil, err
	}

	it.Socketed = append(it.Socketed, filler)

	if len(it.Socketed) == it.Sockets && rw != nil {
		if w := rw.Find(cat, it, ladder); w != nil {
			it.Runeword = w.Name
			return w, nil
		}
	}

	return nil, nil
}

// runewordQuality: a runeword can only form in an item of low, normal or
// superior quality (UNVERIFIED; the usual rule of the game, not traced in the
// binary).
func runewordQuality(q d2drop.Quality) bool {
	return q == 0 || q == d2drop.QualityLow || q == d2drop.QualityNormal || q == d2drop.QualitySuperior
}

// Fits reports whether the runeword may be made in the item: its type is
// included and not excluded. Sockets and runes are checked by Find.
func (w *Runeword) Fits(cat *Catalog, it *Item) bool {
	b, ok := cat.bases[it.Code]
	if !ok {
		return false
	}

	in := false

	for _, t := range w.Include {
		if b.HasType(t) {
			in = true
		}
	}

	if !in {
		return false
	}

	for _, t := range w.Exclude {
		if b.HasType(t) {
			return false
		}
	}

	return true
}

// Find returns the runeword the item's filled sockets spell: every socket
// holds a rune, in the exact order and number of the runeword's runes, the
// runeword is complete (and, when it is ladder-only, the game is a ladder one),
// and the item's type fits.
func (r *Runewords) Find(cat *Catalog, it *Item, ladder bool) *Runeword {
	if it.Sockets == 0 || len(it.Socketed) != it.Sockets || !runewordQuality(it.Quality) {
		return nil
	}

	for _, w := range r.List {
		if !w.Complete || (w.Ladder && !ladder) || len(w.Runes) != len(it.Socketed) {
			continue
		}

		match := true

		for i, code := range w.Runes {
			if !strings.EqualFold(code, it.Socketed[i]) {
				match = false
				break
			}
		}

		if match && w.Fits(cat, it) {
			return w
		}
	}

	return nil
}
