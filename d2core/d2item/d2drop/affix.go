package d2drop

// Affix tables and the affix pickers of Game.exe 1.14b, checked against the
// emulated game (testdata/create_b.json, see create_b_test.go). This replaces
// the unverified affix_legacy.go (still used by diablo2item until it moves to
// the Creator).

// AffixRow is one row of the game's combined affix table: MagicSuffix rows, then
// MagicPrefix rows, then AutoMagic rows (the load order of the game). The
// 1-based position in that table is the affix id stored on items.
type AffixRow struct {
	Name      string
	Version   int
	Spawnable bool
	Rare      bool // may appear on rare, crafted and tempered items
	Level     int
	MaxLevel  int // 0 = none
	Frequency int
	Group     int
	Class     int      // hero class index the affix is restricted to, -1 for none
	IType     []string // item type codes (the first 4 characters, as the game stores them)
	EType     []string
	// Mod1Sockets is true when the affix's first property is the one that sets
	// the number of sockets: such affixes never go on items that cannot be
	// socketed (660b80).
	Mod1Sockets bool
}

// RareName is a row of RarePrefix.txt or RareSuffix.txt.
type RareName struct {
	Name    string
	Version int
	IType   []string
	EType   []string
}

// AffixTables is everything the affix generation reads from the tables.
type AffixTables struct {
	// Rows is the combined table: NSuffix suffixes, NPrefix prefixes, then the
	// automagic rows.
	Rows            []AffixRow
	NSuffix         int
	NPrefix         int
	RareNames       []RareName // rare suffix names first, then rare prefix names
	NRareSuffixName int

	// OnApply, if set, is called when the game applies the properties of an
	// affix (ITEMMODS_ApplyPropertyGroup for an affix row) in the middle of
	// the generation, because that consumes the item generator in order.
	OnApply func(st *itemState, id int)
}

// row returns the affix with the 1-based id, nil if there is none.
func (t *AffixTables) row(id int) *AffixRow {
	if id < 1 || id > len(t.Rows) {
		return nil
	}

	return &t.Rows[id-1]
}

// span returns the half open range of row indexes searched by a pick.
func (t *AffixTables) span(prefix bool, auto int) (lo, hi int) {
	switch {
	case auto != 0:
		return t.NSuffix + t.NPrefix, len(t.Rows)
	case prefix:
		return t.NSuffix, t.NSuffix + t.NPrefix
	default:
		return 0, t.NSuffix
	}
}

func (t *AffixTables) apply(st *itemState, id int) {
	if t.OnApply != nil && id > 0 {
		t.OnApply(st, id)
	}
}

const (
	maxAffixCandidates = 0x1ff // size of the candidate list of the pickers
	maxAffixAlvl       = 99
	classNone          = 7 // 62c210: the class of an item that has none
)

// itemClass is ITEM_GetClassRestriction (62c210): the hero class index of the
// item's type, or 7.
func (c *Creator) itemClass(st *itemState) int {
	if ty := c.Items.Type(st.base); ty != nil && ty.Class >= 0 && ty.Class < classNone {
		return ty.Class
	}

	return classNone
}

// classicExcluded: in classic items (version word below 100) stackable and
// throwable items get no affixes (628bb0 / 62bbd0).
func (c *Creator) classicExcluded(st *itemState) bool {
	if st.req.Version >= 100 {
		return false
	}

	ty := c.Items.Type(st.base)

	return st.base.Stackable || (ty != nil && ty.Throwable)
}

// typeLists is the end of 660b80 and 660c60: the item must not be of any
// etype and must be of one of the itypes; both lists stop at their first
// empty entry.
func (c *Creator) typeLists(st *itemState, itype, etype []string) bool {
	for _, e := range etype {
		if e == "" {
			break
		}

		if c.Items.IsA(st.base, e) {
			return false
		}
	}

	for _, i := range itype {
		if i == "" {
			break
		}

		if c.Items.IsA(st.base, i) {
			return true
		}
	}

	return false
}

// affixTypeOK is ITEMGEN_AffixFitsItem (660b80).
func (c *Creator) affixTypeOK(st *itemState, a *AffixRow) bool {
	if c.classicExcluded(st) {
		return false
	}

	// Affixes that add sockets are only for items that can be socketed.
	if !(st.base.HasInv && c.maxSockets(st) > 0) && a.Mod1Sockets {
		return false
	}

	return c.typeLists(st, a.IType, a.EType)
}

// hasGroup is 5bf160: whether an affix of the group is already on the item.
// Like the game it stops at the first empty slot of each list.
func (c *Creator) hasGroup(st *itemState, group int) bool {
	for _, set := range [2][3]int{st.prefix, st.suffix} {
		for _, id := range set {
			r := c.Affixes.row(id)
			if r == nil {
				break
			}

			if r.Group == group {
				return true
			}
		}
	}

	return false
}

// affixLevel is the affix level (alvl) of the LoD picker (5bf1c0): the larger
// of item level and base level, plus the base's magic level, or lowered by
// half the base level (doubled above the cap); clamped to 1..99.
func affixLevel(ilvl, qlvl, magicLevel int) int {
	v := maxInt(ilvl, qlvl)

	if magicLevel != 0 {
		v += magicLevel
	} else if half := qlvl / 2; v < maxAffixAlvl-half {
		v -= half
	} else {
		v = 2*v - maxAffixAlvl
	}

	return minInt(maxInt(v, 1), maxAffixAlvl)
}

// pickReq are the parameters of the game's affix pickers.
type pickReq struct {
	spawnCheck bool // honour the spawnable column
	force      bool // skip the 50% gate
	apply      bool // apply the affix properties right away
	prefix     bool
	autoGroup  int // automagic group, 0 for normal affixes
}

// pickAffix is ITEMGEN_PickAffixByVersion (5bf590): the Lord of Destruction
// picker for items whose version word is not 0, the classic one otherwise. It
// returns the 1-based affix id, 0 if there is none (or the gate failed). The
// game's forced affix ids (cube recipes) are not supported.
func (c *Creator) pickAffix(st *itemState, r pickReq) int {
	if st.req.Version >= 1 {
		return c.pickAffixLoD(st, r)
	}

	return c.pickAffixClassic(st, r)
}

// gate takes the generator step every picker begins with and reports whether
// the picker goes on.
func (st *itemState) gate(force bool) bool {
	return st.item.Chance() || force
}

type affixCand struct {
	idx int
	row *AffixRow
}

// pickAffixLoD is ITEMGEN_PickAffixLod (5bf1c0).
func (c *Creator) pickAffixLoD(st *itemState, r pickReq) int {
	t := c.Affixes
	lo, hi := t.span(r.prefix, r.autoGroup)

	if lo == hi || !st.gate(r.force) {
		return 0
	}

	alvl := affixLevel(st.ilvl, st.base.Level, st.base.MagicLevel)
	mlvl := st.base.MagicLevel != 0
	class := c.itemClass(st)

	var (
		cands []affixCand
		sum   int
	)

	for i := lo; i < hi; i++ {
		a := &t.Rows[i]

		switch {
		case r.spawnCheck && !a.Spawnable:
			continue
		case a.Version >= 100 && st.req.Version < 100:
			continue
		case a.Level > alvl, a.MaxLevel != 0 && a.MaxLevel < alvl:
			continue
		case !a.Rare && (st.quality == QualityRare || st.quality == QualityCrafted || st.quality == 9):
			continue
		case !c.affixTypeOK(st, a):
			continue
		case r.autoGroup != 0 && r.autoGroup != a.Group:
			continue
		case a.Frequency == 0:
			continue
		case a.Class >= 0 && class != classNone && a.Class != class:
			continue
		case c.hasGroup(st, a.Group):
			continue
		}

		if len(cands) < maxAffixCandidates {
			cands = append(cands, affixCand{i, a})
		}

		if mlvl {
			sum += a.Frequency * a.Level
		} else {
			sum += a.Frequency
		}
	}

	if len(cands) == 0 {
		return 0
	}

	// The roll is walked by subtraction until it goes negative, so the last
	// candidate effectively has one extra weight.
	roll := int(st.item.Roll(int32(sum + 1)))
	pick := cands[len(cands)-1]

	for _, cd := range cands {
		w := cd.row.Frequency
		if mlvl {
			w *= cd.row.Level
		}

		roll -= w
		if roll < 0 {
			pick = cd

			break
		}
	}

	if r.apply {
		t.apply(st, pick.idx+1)
	}

	return pick.idx + 1
}

// pickAffixClassic is ITEMGEN_PickAffixClassic (5bef10): level ilvl+2, no
// frequency, no maximum level, no group or class tests, uniform pick.
func (c *Creator) pickAffixClassic(st *itemState, r pickReq) int {
	t := c.Affixes
	lo, hi := t.span(r.prefix, r.autoGroup)

	if lo == hi || !st.gate(r.force) {
		return 0
	}

	alvl := minInt(maxInt(st.ilvl+2, 1), maxAffixAlvl)

	var cands []int

	for i := lo; i < hi; i++ {
		a := &t.Rows[i]

		switch {
		case r.spawnCheck && !a.Spawnable:
			continue
		case a.Version >= 100 && st.req.Version < 100:
			continue
		case a.Level > alvl:
			continue
		case !c.affixTypeOK(st, a):
			continue
		}

		if len(cands) < maxAffixCandidates {
			cands = append(cands, i)
		}
	}

	if len(cands) == 0 {
		return 0
	}

	idx := cands[st.item.Roll(int32(len(cands)))]

	if r.apply {
		t.apply(st, idx+1)
	}

	return idx + 1
}

// nameFits is ITEMGEN_RareNameFits (660c60).
func (c *Creator) nameFits(st *itemState, n *RareName) bool {
	switch {
	case c.classicExcluded(st):
		return false
	case n.Version >= 100 && st.req.Version < 100:
		return false
	}

	return c.typeLists(st, n.IType, n.EType)
}

// pickRareName is ITEMGEN_PickRareName (5bf770 / 5bf650, identical): a uniform
// pick among the names that fit the item, from the prefix or the suffix
// names. It returns the 1-based id in the table of names (suffix names come
// first), 0 if none fits.
func (c *Creator) pickRareName(st *itemState, prefix bool) int {
	t := c.Affixes
	lo, hi := 0, t.NRareSuffixName

	if prefix {
		lo, hi = t.NRareSuffixName, len(t.RareNames)
	}

	var cands []int

	for i := lo; i < hi; i++ {
		if c.nameFits(st, &t.RareNames[i]) && len(cands) < maxAffixCandidates {
			cands = append(cands, i)
		}
	}

	if len(cands) == 0 {
		return 0
	}

	return cands[st.item.Roll(int32(len(cands)))] + 1
}
