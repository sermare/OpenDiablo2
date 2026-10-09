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

		switch {
		case it.Weapon != nil:
			out = append(out, g.Weapon...)
		case it.BaseBlock > 0:
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
	Pieces  int
	Partial [][]Prop // Partial[i] applies from i+2 worn pieces (PCode2a/2b ...)
	Full    []Prop   // applies when all pieces are worn (FCode1-8)
}

// SetDefs implements SetTable over expanded definitions keyed by set id.
type SetDefs map[int]SetDef

// SetBonus implements SetTable.
func (s SetDefs) SetBonus(setID, n int) []Prop {
	d, ok := s[setID]
	if !ok {
		return nil
	}

	var out []Prop

	for i, props := range d.Partial {
		if n >= i+2 {
			out = append(out, props...)
		}
	}

	if d.Pieces > 0 && n >= d.Pieces {
		out = append(out, d.Full...)
	}

	return out
}
