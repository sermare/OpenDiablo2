package d2statlist

// GemDef holds what a gem or rune does in each kind of socket (Gems.txt).
type GemDef struct {
	Weapon, Helm, Shield []Prop
}

// GemTable maps a gem/rune item code to its effects.
type GemTable map[string]GemDef

// gemProps returns the properties of the plain gems and runes socketed into
// it. Runeword items take their effect from RunewordProps (the runes inside
// carry none), jewels from their own Props, so those are skipped here.
// Armor and helms use the "helm" column, shields the "shield" column,
// weapons the "weapon" column (Gems.txt).
func (e *Env) gemProps(it *Item) []Prop {
	if e.Gems == nil || it.Runeword {
		return nil
	}

	var out []Prop

	for _, s := range it.Sockets {
		g, ok := e.Gems[s.Code]
		if !ok {
			continue
		}

		switch it.gemColumn() {
		case 1:
			out = append(out, g.Weapon...)
		case 3:
			out = append(out, g.Shield...)
		default:
			out = append(out, g.Helm...)
		}
	}

	return out
}

// SetTable resolves the bonuses of a set.
type SetTable interface {
	// SetBonus returns the set's bonus properties when n of its pieces are
	// worn (partial tiers up to n, and the full bonus when n is the whole set).
	SetBonus(setID, n int) []Prop
}

// SetDef is one set from Sets.txt with its properties already expanded.
type SetDef struct {
	Pieces int
	// Members are the SetItems rows of the set in table order (the position
	// is the piece index of the game) and AddFuncs their "add func" column.
	// Both optional: without them every worn set item counts as its own piece.
	Members  []int
	AddFuncs []int
	Partial  [][]Prop // Partial[i] applies from i+2 worn pieces (PCode2a/2b ...)
	Full     []Prop   // applies when all pieces are worn (FCode1-8)
}

// SetDefs implements SetTable over expanded definitions keyed by set id.
type SetDefs map[int]SetDef

// pieceInfo returns the position in the set and the add func of the set item
// with the given SetItems row.
func (s SetDefs) pieceInfo(setID, row int) (idx, addFunc int, ok bool) {
	d, found := s[setID]
	if !found {
		return 0, 0, false
	}

	for i, m := range d.Members {
		if m == row {
			if i < len(d.AddFuncs) {
				addFunc = d.AddFuncs[i]
			}

			return i, addFunc, true
		}
	}

	return 0, 0, false
}

// SetBonus implements SetTable.
//
// VERIFIED (ITEM_ApplySetBonusProperties 6626a0): with n worn distinct pieces
// the partial tiers 2 .. min(n, pieces-1) apply (the tier of the last piece
// never does: the full bonus takes over), and the full bonus applies from
// n >= pieces. The piece count is VERIFIED too: DATATBL_LoadSetsTable 635680
// counts the SetItems rows of each set into field +0xc (rows past 6 are
// ignored), and the worn count is the popcount of the mate bit mask (table
// 6eeff0). The shipped data fills the last tier of 3 and 4 piece sets but the
// game never uses it.
func (s SetDefs) SetBonus(setID, n int) []Prop {
	d, ok := s[setID]
	if !ok {
		return nil
	}

	var out []Prop

	top := n
	if d.Pieces > 0 && top > d.Pieces-1 {
		top = d.Pieces - 1
	}

	for i, props := range d.Partial {
		if top >= i+2 {
			out = append(out, props...)
		}
	}

	if d.Pieces > 0 && n >= d.Pieces {
		out = append(out, d.Full...)
	}

	return out
}

// gemColumn returns 1 weapon, 2 armor/helm, 3 shield: the item's gemapplytype
// plus one. VERIFIED as the only input of the exe (INV_ServerApplyEquippedItemEffects
// 55a080 passes ITEM_GetGemApplyType 629c10, the host base's byte, to the
// property group). Without it the old guess is used (UNVERIFIED; it misses the
// Buckler, a shield with block 0).
func (it *Item) gemColumn() int {
	switch {
	case it.GemKind > 0:
		return it.GemKind
	case it.Weapon != nil:
		return 1
	case it.BaseBlock > 0:
		return 3
	}

	return 2
}
