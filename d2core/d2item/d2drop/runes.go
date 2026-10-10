package d2drop

import "strings"

// Runewords (Runes.txt) and set bonuses (Sets.txt).
//
// VERIFIED (Ghidra, ITEM_ApplyRunewordProperties 0x662620): the walk finds the
// runeword of the item's sockets, does nothing if the item already has a stat
// list 0xab, sets item flag 0x4000000, then applies the runeword's properties
// one after the other (at most seven, stopping at the first empty one) as
// property kind 6 into the stat list 0xab.
//
// VERIFIED (Ghidra, ITEM_FindRunewordForSockets 0x62c010): the search gives up
// for an item of quality 4 to 9 (magic, set, rare, unique, crafted, tempered:
// only low quality, normal and superior items make runewords), for a quest
// item, and when the number of socketed items differs from the item's socket
// count (every socket must be filled). Then the first row of Runes.txt in
// table order wins that is flagged complete, whose runes (item classes,
// stopping at the first empty one) equal the socketed items in socket order
// (a longer row never matches), none of whose etypes the item is of
// (ITEM_IsOfType, which follows the type inheritance) and of at least one of
// whose itypes it is.

const (
	// ListRuneword is the selector of the stat list of a runeword.
	ListRuneword = 0xab
	// FlagRuneword is the item flag of a runeword item.
	FlagRuneword uint32 = 0x4000000

	maxRunewordProps = 7
)

// Runeword is a row of Runes.txt.
type Runeword struct {
	Name     string
	Complete bool
	IType    []string
	EType    []string
	Runes    []string // misc.txt codes of the runes, socket order
	Props    [maxRunewordProps]PropInst
}

// RuneTables holds Runes.txt.
type RuneTables struct {
	Words []Runeword
}

// FindRuneword returns the runeword made by the runes (item codes) socketed in
// an item of the base code, nil if there is none.
//
// It applies the item rules of the exe for a normal quality item whose sockets
// are all filled; FindRunewordFor takes the quality and the socket count.
func (c *Creator) FindRuneword(base string, sockets []string) *Runeword {
	return c.FindRunewordFor(base, QualityNormal, len(sockets), sockets)
}

// FindRunewordFor is FindRuneword for an item of a quality with a number of
// sockets (see the rules above).
func (c *Creator) FindRunewordFor(base string, quality Quality, socketCount int, sockets []string) *Runeword {
	b := c.Items.ByCode[base]
	if b == nil || c.Runes == nil || len(sockets) == 0 || b.Quest != 0 ||
		(quality >= QualityMagic && quality <= qualityTempered) || socketCount != len(sockets) {
		return nil
	}

	for i := range c.Runes.Words {
		w := &c.Runes.Words[i]
		if !w.Complete || len(w.Runes) != len(sockets) {
			continue
		}

		same := true

		for k := range sockets {
			if w.Runes[k] != strings.TrimSpace(sockets[k]) {
				same = false
			}
		}

		if !same || !c.typeFits(b, w.IType, w.EType) {
			continue
		}

		return w
	}

	return nil
}

func (c *Creator) typeFits(b *BaseItem, include, exclude []string) bool {
	fits := false

	for _, t := range include {
		if t != "" && c.Items.IsA(b, t) {
			fits = true
		}
	}

	for _, t := range exclude {
		if t != "" && c.Items.IsA(b, t) {
			return false
		}
	}

	return fits
}

// ApplyRuneword walks the properties of a runeword over an item that was
// created as r (a copy with the runeword's stats added is returned). The
// request is the one the item was created with. An item that already has the
// runeword's stat list is returned unchanged, like the game does.
func (c *Creator) ApplyRuneword(r *Rolled, req Request, w *Runeword) *Rolled {
	for _, wr := range r.Writes {
		if wr.List == ListRuneword {
			return r
		}
	}

	base := c.Items.ByCode[req.Code]
	if base == nil || w == nil {
		return r
	}

	st := &itemState{
		c: c, req: req, base: base, ilvl: r.ILvl, quality: r.Quality, flags: r.Flags | FlagRuneword,
		unit: r.UnitSeed, item: r.ItemSeed, uniqueRow: r.Unique,
		writes: append([]StatWrite(nil), r.Writes...),
	}

	for i := 0; i < maxRunewordProps; i++ {
		if w.Props[i].Prop < 0 {
			break
		}

		c.applyInstance(st, PropKindSocket6, w.Props[i], ListRuneword)
	}

	out := *r
	out.Flags = st.flags
	out.Writes = st.writes
	out.UnitSeed, out.ItemSeed = st.unit, st.item

	return &out
}

// SetBonus is the bonus of one set: Partial[i] is active from i+2 pieces worn
// (PCode2a/2b ... PCode5a/5b), Full when every piece is worn (FCode1..8). The
// values are the stat writes of the property engine (shifted by valshift).
type SetBonus struct {
	Pieces  int
	Partial [4][]StatWrite
	Full    []StatWrite
}

// SetBonuses rolls the bonuses of the set with the row of Sets.txt through the
// property engine (set bonuses have fixed values, min equals max).
func (c *Creator) SetBonuses(set int) (SetBonus, bool) {
	if c.Uniques == nil || set < 0 || set >= len(c.Uniques.Sets) {
		return SetBonus{}, false
	}

	def := &c.Uniques.Sets[set]
	out := SetBonus{Pieces: len(def.Items)}

	run := func(insts []PropInst) []StatWrite {
		st := &itemState{c: c, req: Request{Version: lodVersionNumber, Expansion: true}, base: &BaseItem{}, ilvl: 1}

		for i := range insts {
			c.applyInstance(st, PropKindSocket6, insts[i], 0)
		}

		return st.writes
	}

	for t := 0; t < 4; t++ {
		out.Partial[t] = run(def.Partial[t*2 : t*2+2])
	}

	out.Full = run(def.Full[:])

	return out, true
}

const lodVersionNumber = 100
