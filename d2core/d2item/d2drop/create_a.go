package d2drop

// Slice A: base stats, quality application glue, low/superior quality,
// ethereal, sockets. Ported from Game.exe 1.14b and proven equal to it by the
// emulator goldens (testdata/create_a.json): ITEMGEN_CreateItemFromRequest
// 556e00 minus the unit creation, InitItemBaseStats 555b10,
// ApplyQualityToItem 555520, RollExistingItemQuality 555050, RollSockets
// 554c60, RollEthereal 554d90 (+ 660a40), ApplySuperiorQuality 5c0710 (with
// the type test 660d10), ApplyLowQuality 5c0b00 (Lord of Destruction) and
// 5c0890 (classic), the normal-quality tails 554f70 / 554e70 and the class
// skill roll 5beea0 (5bebc0 / 5be9a0) of staves, orbs, pelts and friends.
//
// Statements marked UNVERIFIED are not reached by the goldens.

// Item flags of pItemData+0x18 that slice A sets.
const (
	flagIdentified uint32 = 0x10
	flagSocketed   uint32 = 0x800
	flagEthereal   uint32 = 0x400000
	flagCreated    uint32 = 0x80000 // set on every item by 556e00
	flagNewItem    uint32 = 0x2000  // set by 556e00 when the request has no saved item data
)

// Request flags the quality code reads beyond the exported ones (bit meanings
// are the game's; the names describe what slice A uses them for).
const (
	// FlagSkillBoost makes the class skill roll of staves use the item level
	// as its bonus (request flags & 0x20 in 5beea0).
	FlagSkillBoost RequestFlags = 0x20
	// FlagSuperiorFallback makes a failed quality roll end in superior
	// instead of low quality (request flags & 0x40 in 555050).
	FlagSuperiorFallback RequestFlags = 0x40
)

// Stats slice A writes (ItemStatCost ids).
const (
	statGold       = 0x0e
	statBlock      = 0x14
	statMinDam     = 0x15
	statMaxDam     = 0x16
	statSecMinDam  = 0x17
	statSecMaxDam  = 0x18
	statDefense    = 0x1f
	statArmorSpeed = 0x43
	statWeapSpeed  = 0x44
	statQuantity   = 0x46
	statElixir     = 0x47
	statDurCur     = 0x48
	statDurMax     = 0x49
	statNoDurab    = 0x98
	statThrowMin   = 0x9f
	statThrowMax   = 0xa0
	statSockets    = 0xc2
	statSkillBonus = 0x6b
	statDifficulty = 0x164
)

// SuperiorRow is a row of QualityItems.txt reduced to the item type columns
// (the stat columns are slice C's: they are applied by the property engine).
type SuperiorRow struct {
	Armor, Weapon, Shield, Scepter, Wand, Staff, Bow, Boots, Gloves, Belt bool
}

// RatioRow is a row of ItemRatio.txt.
type RatioRow struct {
	Version       int
	Uber          bool
	ClassSpecific bool
	Ratio         Ratio
}

// BookRow is a row of Books.txt: the codes of the scroll and of the book of
// a spell. Scrolls and tomes store their row (the spell) in their first
// suffix slot.
type BookRow struct {
	Scroll, Book string
}

// QualityTables holds QualityItems.txt and LowQualityItems.txt, plus the
// small tables the same code paths read: ItemRatio.txt (quality of shop and
// gamble items), Books.txt and the class skill lists of Skills.txt.
type QualityTables struct {
	Superior []SuperiorRow // QualityItems.txt, in file order
	Low      int           // number of LowQualityItems.txt rows
	Ratios   []RatioRow
	Books    []BookRow
	// ClassSkills lists the skill ids of each hero class (amazon 0 ...
	// assassin 6) in Skills.txt order; SkillType is the item type a skill
	// requires (Skills.txt itypea1), by skill id.
	ClassSkills [7][]int
	SkillType   map[int]string
}

// ratio is 6386f0: the ItemRatio row of the matching class-specific / uber
// kind with the highest version not above the game's (0 classic, 100 LoD).
func (q *QualityTables) ratio(classSpecific, uber bool, lod bool) *Ratio {
	ver := 0
	if lod {
		ver = 100
	}

	var best *RatioRow

	for i := range q.Ratios {
		r := &q.Ratios[i]
		if r.ClassSpecific != classSpecific || r.Uber != uber || r.Version > ver {
			continue
		}

		if best == nil || r.Version > best.Version {
			best = r
		}
	}

	if best == nil {
		return nil
	}

	return &best.Ratio
}

// heroClass maps the class codes of ItemTypes.txt and Skills.txt to indices.
func heroClass(code string) int {
	for i, c := range []string{"ama", "sor", "nec", "pal", "bar", "dru", "ass"} {
		if c == code {
			return i
		}
	}

	return -1
}

// ---------------------------------------------------------------------------
// item state helpers

func (c *Creator) itype(st *itemState) *ItemType {
	if t := c.Items.Type(st.base); t != nil {
		return t
	}

	return &ItemType{}
}

func (c *Creator) isA(st *itemState, code string) bool { return c.Items.IsA(st.base, code) }

// stat is the unit's own value of a stat (the last one set).
func (st *itemState) stat(id int) int {
	for i := len(st.writes) - 1; i >= 0; i-- {
		if w := st.writes[i]; w.Kind == 'S' && w.Stat == id {
			return w.Value
		}
	}

	return 0
}

func (st *itemState) setStat(id, v int) { st.write('S', id, v, 0) }

// durable is 629b00: the item has a durability that can be changed.
func (st *itemState) durable() bool {
	return !st.base.NoDurability && st.base.Durability > 0 && st.stat(statNoDurab) <= 0
}

// ---------------------------------------------------------------------------
// creation

// createItem runs the creation after the units exist: base stats, then the
// quality application (the part of ITEMGEN_CreateItemFromRequest after the
// unit has been created).
func (c *Creator) createItem(st *itemState) error {
	b := st.base
	// "play" is the ear: its player name and level come from the dropper,
	// which the oracle cannot create. UNVERIFIED, not created here.
	if b.Type == "play" {
		return ErrNotCreated
	}

	// 556e00 raises the request's item level to 1 in place.
	if st.req.ILvl < 1 {
		st.req.ILvl = 1
	}

	st.itemWord = st.item.Lo
	st.flags = flagIdentified | flagCreated | flagNewItem
	st.uniqueRow = 0

	c.initBaseStats(st)

	ok := c.applyQuality(st)

	if b.Quest != 0 && b.QuestDiffCheck != 0 {
		// Quest items remember the difficulty they were created in.
		st.write('L', statDifficulty, st.req.Difficulty, 0)

		st.flags |= flagIdentified
	}

	if !ok {
		return ErrNotCreated
	}

	return nil
}

// unitRoll is rand(n) on the unit generator (base stats).
func (st *itemState) unitRoll(n int) int { return int(st.unit.Roll(int32(n))) }

// itemRoll is rand(n) on the item generator (affixes, sockets).
func (st *itemState) itemRoll(n int) int { return int(st.item.Roll(int32(n))) }

// rollDurability is the shared durability roll of weapons and armor: the
// current durability is half the maximum plus up to the other half.
func (st *itemState) rollDurability() {
	max := st.base.Durability
	half := max >> 1

	cur := st.unitRoll(half) + half
	if cur >= 0xff {
		cur = 0xff
	}

	st.setStat(statDurCur, cur)

	if max >= 0xff {
		max = 0xff
	}

	st.setStat(statDurMax, max)
}

// rollQuantity is the stack size roll of stackable items.
func (c *Creator) rollQuantity(st *itemState, weapon bool) {
	b := st.base
	min := b.MinStack
	max := b.MaxStack + st.stat(0xfe)

	if max > 0x1ff {
		max = 0x1ff
	}

	var n int

	if weapon {
		n = st.unitRoll(max-min) + min
		if n < 1 {
			n = 1
		}
	} else {
		limit := b.SpawnStack
		if limit < min || limit == 0 {
			limit = max
			if min > max {
				limit = min
			}
		}

		n = st.unitRoll(limit-min) + min
		if n < 1 {
			n = 1
		}
	}

	// The game forces 1 for items whose quality is already magical or
	// better; the unit's quality is still the default (normal) here.
	st.setStat(statQuantity, n)
}

// initBaseStats is ITEMGEN_InitItemBaseStats (555b10).
func (c *Creator) initBaseStats(st *itemState) {
	b := st.base
	ty := c.itype(st)

	switch {
	case b.Type == "gold":
		amount := st.unitRoll(st.req.ILvl*5) + st.req.ILvl
		if amount <= 0 {
			amount = 1
		}

		st.setStat(statGold, amount)
	case ty.Quiver:
		c.rollQuantity(st, true)
		// Quivers have no durability and no VarInvGfx.
	case c.isA(st, "armo"):
		st.setStat(statBlock, b.Block)
		st.setStat(statArmorSpeed, -b.Speed)
		st.rollDurability()

		lo, hi := b.MinAC, b.MaxAC
		def := lo
		if n := hi - lo + 1; n > 0 {
			def = st.unitRoll(n) + lo
		}

		st.setStat(statDefense, def)
	case c.isA(st, "weap"):
		if b.Stackable {
			c.rollQuantity(st, true)
		}

		st.rollDurability()
		c.weaponDamage(st)
		st.setStat(statWeapSpeed, -b.Speed)
	default:
		if b.Stackable {
			c.rollQuantity(st, false)
		}

		if b.Type == "elix" {
			c.elixir(st)
		}
	}

	if n := ty.VarInvGfx; n != 0 {
		st.gfx[1] = st.unitRoll(n)
	}
}

// weaponDamage is 554500: the base damage of the weapon table as stats.
func (c *Creator) weaponDamage(st *itemState) {
	b := st.base

	if b.MaxDam != 0 {
		st.setStat(statMaxDam, b.MaxDam)
	}

	if b.MinDam != 0 {
		st.setStat(statMinDam, b.MinDam)
	}

	if b.TwoHandMinDam != 0 {
		st.setStat(statSecMinDam, b.TwoHandMinDam)
	}

	if b.TwoHandMaxDam != 0 {
		st.setStat(statSecMaxDam, b.TwoHandMaxDam)
	}

	if b.MaxMisDam != 0 {
		st.setStat(statThrowMin, b.MinMisDam)
		st.setStat(statThrowMax, b.MaxMisDam)
	}
}

// elixirTable is the table at 0x7426b8 of the game (6 entries, 660e40).
var elixirTable = [6]int{0, 1, 2, 3, 9, 7}

// elixir is 660e40, done for the item type 11 ("elix").
func (c *Creator) elixir(st *itemState) {
	pick := elixirTable[st.itemRoll(len(elixirTable))]
	st.uniqueRow = pick

	if pick == 9 || pick == 7 {
		st.setStat(statElixir, (st.itemRoll(4)+1)<<8)
	} else {
		st.setStat(statElixir, 1)
	}
}

// ---------------------------------------------------------------------------
// quality application

// resetAffixes is 555330 (and the start of 555520): no affixes, no names, no
// unique row.
func (st *itemState) resetAffixes() {
	st.prefix, st.suffix = [3]int{}, [3]int{}
	st.rareNames = [2]int{}
	st.uniqueRow = -1
}

// rollExistingQuality is ITEMGEN_RollExistingItemQuality (555050): the quality
// of an item that did not come from a treasure class.
func (c *Creator) rollExistingQuality(st *itemState) Quality {
	lod := st.req.Version >= 1
	if lod && st.req.Quality != 0 {
		return st.req.Quality
	}

	b := st.base
	ty := c.itype(st)

	types := c.typeCodes(b)
	uber := UberTier(b.Code, b.UberCode, b.UltraCode, b.Type, types, b.Quest != 0)

	// ClassSpecific: the item's primary type is restricted to a hero class.
	r := c.Quality.ratio(ty.Class >= 0, uber, st.req.Version >= 100)
	if r == nil {
		// The game stops with an error here (a classic game has no
		// class specific ItemRatio rows); report no quality.
		return QualityNone
	}

	if b.Quest != 0 {
		return QualityNormal
	}

	d := st.req.ILvl

	if lod {
		if c.isA(st, "misc") {
			d = 1
		} else {
			d -= b.Level
			if d < 1 {
				d = 1
			}
		}
	}

	// The tiers in the game's order; classic uses a plain subtraction for
	// unique, rare and set.
	type tier struct {
		q       Quality
		dr      DropRatio
		minusLv bool
	}

	tiers := []tier{
		{QualityUnique, r.Unique, true}, {QualityRare, r.Rare, true}, {QualitySet, r.Set, true},
		{QualityMagic, r.Magic, false}, {QualitySuperior, r.HiQuality, false}, {QualityNormal, r.Normal, false},
	}

	for _, t := range tiers {
		var v int

		if lod || !t.minusLv {
			v = t.dr.Base - div(d, t.dr.Divisor)
		} else {
			v = t.dr.Base - d
		}

		if v <= 0 {
			if lod {
				return t.q
			}

			v = 1
		}

		if st.itemRoll(v) == 0 {
			return t.q
		}
	}

	if st.req.Flags&FlagSuperiorFallback != 0 {
		return QualitySuperior
	}

	return QualityLow
}

// typeCodes is the item type and its ancestors, type2 included.
func (c *Creator) typeCodes(b *BaseItem) []string {
	var out []string

	seen := map[string]bool{}

	for _, code := range []string{b.Type, b.Type2} {
		if t := c.Items.Types[code]; t != nil {
			for _, a := range t.Ancestors {
				if !seen[a] {
					seen[a] = true
					out = append(out, a)
				}
			}
		}
	}

	return out
}

// applyQuality is ITEMGEN_ApplyQualityToItem (555520).
func (c *Creator) applyQuality(st *itemState) bool {
	b := st.base
	ty := c.itype(st)

	st.auto = 0
	st.prefix, st.suffix = [3]int{}, [3]int{}
	st.rareNames = [2]int{}

	q := c.rollExistingQuality(st)
	if q == QualityNone {
		return false
	}

	if st.req.Quality != 0 {
		q = st.req.Quality
	}

	st.quality = q

	if ty.Magic {
		if b.Quest != 0 {
			st.quality = QualityUnique
		} else if st.quality < QualityMagic || st.quality > 9 {
			st.quality = QualityMagic
		}
	}

	if !ty.Rare && st.quality == QualityRare {
		st.quality = QualityMagic
	}

	if b.Unique {
		st.quality = QualityUnique
	}

	if ty.Normal {
		st.quality = QualityNormal
	}

	var saved uint32

	save := func() { saved = st.item.Lo }

	switch st.quality {
	case QualityLow:
		save()

		if !c.applyLow(st) {
			c.fallbackNormal(st, &saved)
		}
	case QualityNormal:
		c.normalQuality(st)
	case QualitySuperior:
		st.uniqueRow = -1

		save()

		if !c.applySuperior(st) {
			c.fallbackNormal(st, &saved)
		}
	case QualityMagic:
		save()

		if !c.rollMagic(st) && !c.fallbackSuperior(st, &saved) {
			c.fallbackNormal(st, &saved)
		}
	case QualitySet:
		st.uniqueRow = -1

		save()

		if !c.pickSet(st) {
			if st.durable() && st.req.Version >= 1 {
				st.scaleDurability(2)
			}

			c.fallbackChain(st, &saved)
		}
	case QualityRare:
		st.rareNames = [2]int{}

		save()

		if !c.rollRare(st) {
			c.fallbackChain(st, &saved)
		}
	case QualityUnique:
		st.uniqueRow = -1

		save()

		if !c.pickUnique(st) {
			if st.durable() && st.req.Version >= 1 {
				st.scaleDurability(3)
			}

			if !c.fallbackRare(st, &saved) {
				c.fallbackChain(st, &saved)
			}
		}
	case QualityCrafted:
		st.rareNames = [2]int{}

		save()

		if !c.rollCrafted(st) {
			c.fallbackNormal(st, &saved)
		}
	case 9: // tempered
		st.rareNames = [2]int{}

		save()

		if !c.rollTempered(st) {
			c.fallbackNormal(st, &saved)
		}
	}

	q = st.quality
	if q <= 0 || q >= 10 {
		return false
	}

	lod := st.req.Version >= 100

	if lod {
		c.rollEthereal(st)
	}

	if q <= QualitySuperior {
		c.rollSockets(st)
	}

	if lod && q != QualitySet && q != QualityUnique && b.AutoPrefix != 0 {
		if id := c.pickAutomagic(st); id > 0 {
			st.auto = id
			// The automagic properties are applied by slice C.
		}
	}

	return true
}

// scaleDurability multiplies the current and maximum durability, capped.
func (st *itemState) scaleDurability(f int) {
	cur := st.stat(statDurCur) * f
	if cur >= 0xff {
		cur = 0xff
	}

	st.setStat(statDurCur, cur)

	max := st.stat(statDurMax) * f
	if max >= 0xff {
		max = 0xff
	}

	st.setStat(statDurMax, max)
}

// retry re-seeds the item generator from a saved word and rolls the
// quality again, the common start of every fallback (555330 + 628110 +
// 652300 + 555050).
func (c *Creator) retry(st *itemState, saved *uint32, q Quality) {
	st.resetAffixes()
	st.itemWord = *saved
	st.item.Init(*saved)

	c.rollExistingQuality(st)

	st.quality = q
	st.req.Quality = q
}

// fallbackNormal is 555380: the item becomes normal.
func (c *Creator) fallbackNormal(st *itemState, saved *uint32) {
	c.retry(st, saved, QualityNormal)
	c.normalQuality(st)
}

// fallbackSuperior is 5553f0.
func (c *Creator) fallbackSuperior(st *itemState, saved *uint32) bool {
	c.retry(st, saved, QualitySuperior)

	*saved = st.item.Lo

	return c.applySuperior(st)
}

// fallbackMagic is 555450.
func (c *Creator) fallbackMagic(st *itemState, saved *uint32) bool {
	c.retry(st, saved, QualityMagic)

	*saved = st.item.Lo

	return c.rollMagic(st)
}

// fallbackRare is 5554c0.
func (c *Creator) fallbackRare(st *itemState, saved *uint32) bool {
	c.retry(st, saved, QualityRare)

	*saved = st.item.Lo

	return c.rollRare(st)
}

// fallbackChain is the end of the set, rare and unique cases (55597a):
// magic, then superior, then normal.
func (c *Creator) fallbackChain(st *itemState, saved *uint32) {
	if c.fallbackMagic(st, saved) {
		return
	}

	if c.fallbackSuperior(st, saved) {
		return
	}

	c.fallbackNormal(st, saved)
}

// ---------------------------------------------------------------------------
// normal quality (555020)

func (c *Creator) normalQuality(st *itemState) {
	if st.req.Version < 1 {
		c.normalClassic(st)
	} else {
		c.normalLoD(st)
	}
}

// bookRow is 5c02d0: the Books.txt row of a scroll (or book) item, or the
// number of rows when there is none.
func (c *Creator) bookRow(st *itemState, scroll bool) int {
	for i, r := range c.Quality.Books {
		code := r.Book
		if scroll {
			code = r.Scroll
		}

		if code == st.base.Code {
			return i
		}
	}

	return len(c.Quality.Books)
}

// normalLoD is 554f70.
func (c *Creator) normalLoD(st *itemState) {
	// Charms, body parts and ears are handled by slices B and the dropper
	// (UNVERIFIED here: charms are always magic and never come this way).
	if c.isA(st, "scro") {
		st.suffix[0] = c.bookRow(st, true)
	}

	if c.isA(st, "book") {
		st.suffix[0] = c.bookRow(st, false)
	}

	c.staffMods(st)
}

// normalClassic is 554e70: the classic game branches on the exact type.
func (c *Creator) normalClassic(st *itemState) {
	switch st.base.Type {
	case "book":
		st.suffix[0] = c.bookRow(st, false)
	case "scro":
		st.suffix[0] = c.bookRow(st, true)
	case "play", "body", "char":
		// UNVERIFIED: ears, body parts and charms never reach this.
	default:
		c.staffMods(st)
		c.rollSockets(st)
	}
}

// ---------------------------------------------------------------------------
// class skill bonus of staves and the like (5beea0)

func (c *Creator) staffMods(st *itemState) {
	cls := heroClass(c.itype(st).StaffMods)
	if cls < 0 || len(c.Quality.ClassSkills[cls]) == 0 {
		return
	}

	first := c.Quality.ClassSkills[cls][0]
	ilvl := st.ilvl

	switch {
	case st.req.Version < 1:
		c.skillsClassic(st, ilvl, first)
	case st.req.Flags&FlagSkillBoost != 0:
		c.skillsLoD(st, ilvl, first, ilvl)
	default:
		c.skillsLoD(st, ilvl, first, 0)
	}
}

// skillTier is the highest skill row tier an item level opens.
func skillTier(ilvl int, lod bool) int {
	switch {
	case lod && ilvl > 0x24:
		return 5
	case ilvl > 0x18:
		return 4
	case ilvl > 0x12:
		return 3
	case ilvl > 0xb:
		return 2
	}

	return 1
}

// tierJitter moves the tier by the d100 roll: +1 above 80, -1 or -2 up to 30.
func tierJitter(tier, r int) int {
	switch {
	case r > 80:
		tier++
	case r <= 30:
		if r > 10 {
			tier--
		} else {
			tier -= 2
		}
	}

	if tier < 1 {
		tier = 1
	}

	return tier
}

// skillsClassic is 5be9a0.
func (c *Creator) skillsClassic(st *itemState, ilvl, first int) {
	r := st.itemRoll(100)

	var n int

	switch {
	case r > 90:
		n = 3
	case r > 70:
		n = 2
	case r > 30:
		n = 1
	default:
		return
	}

	tier := skillTier(ilvl, false)
	chosen := [3]int{-1, -1, -1}

	for i := 0; i < n; i++ {
		t := tierJitter(tier, st.itemRoll(100))

		var skill int

		for {
			v := int(st.item.Step())
			skill = v%5 + 5*(t-1) + first

			if skill != 0x49 && skill != chosen[0] && skill != chosen[1] && skill != chosen[2] {
				break
			}
		}

		chosen[i] = skill

		value := 1

		if r3 := st.itemRoll(100); r3 >= 90 {
			value = 3
		} else if r3 >= 60 {
			value = 2
		}

		st.write('L', statSkillBonus, value, skill)
	}
}

// skillsLoD is 5bebc0.
func (c *Creator) skillsLoD(st *itemState, ilvl, first, boost int) {
	r := st.itemRoll(100) + boost

	var n int

	switch {
	case r > 90:
		n = 3
	case r > 70:
		n = 2
	case r > 30:
		n = 1
	default:
		if boost == 0 {
			return
		}

		n = 1
	}

	tier := skillTier(ilvl, st.req.Version >= 100)
	chosen := [3]int{-1, -1, -1}

	for i := 0; i < n; i++ {
		t := tierJitter(tier, st.itemRoll(100))
		if st.quality == QualityLow && t >= 4 {
			t = 4
		}

		var skill int

		for tries := 6; ; {
			v := int(st.item.Step())
			skill = v%5 + 5*(t-1) + first

			ok := skill != chosen[0] && skill != chosen[1] && skill != chosen[2]
			if need, has := c.Quality.SkillType[skill]; has && !c.isA(st, need) {
				ok = false
			}

			if ok {
				chosen[i] = skill
				break
			}

			tries--
			if tries <= 0 {
				break
			}
		}

		value := 1

		if !(st.req.Version >= 100 && st.quality == QualityLow) {
			if r3 := st.itemRoll(100) + boost/2; r3 >= 90 {
				value = 3
			} else if r3 >= 60 {
				value = 2
			}
		}

		st.write('L', statSkillBonus, value, skill)
	}
}

// ---------------------------------------------------------------------------
// superior quality (5c0710)

// superiorFits is 660d10: whether a QualityItems row may apply to the item.
func (c *Creator) superiorFits(st *itemState, r SuperiorRow) bool {
	t := st.base.Type

	if r.Weapon && c.isA(st, "weap") {
		switch t {
		case "staf", "bow", "xbow", "scep", "wand":
		default:
			return true
		}
	}

	if r.Armor && c.isA(st, "armo") {
		switch t {
		case "shie", "boot", "glov", "belt":
		default:
			return true
		}
	}

	for _, p := range []struct {
		code string
		flag bool
	}{
		{"shie", r.Shield}, {"scep", r.Scepter}, {"wand", r.Wand}, {"staf", r.Staff},
		{"bow", r.Bow}, {"xbow", r.Bow}, {"boot", r.Boots}, {"glov", r.Gloves}, {"belt", r.Belt},
	} {
		if t == p.code && p.flag {
			return true
		}
	}

	return false
}

// applySuperior is ITEMGEN_ApplySuperiorQuality: it picks a row of
// QualityItems.txt the item can take. The row's properties are slice C's.
func (c *Creator) applySuperior(st *itemState) bool {
	rows := c.Quality.Superior
	n := len(rows)

	if c.itype(st).Throwable || st.base.NoDurability {
		n = 4
	}

	tried := make([]bool, len(rows)+8)

	for {
		idx := 0
		if n > 0 {
			idx = st.itemRoll(n)
		}

		if tried[idx] {
			continue
		}

		if idx >= len(rows) {
			return false
		}

		if c.superiorFits(st, rows[idx]) {
			st.uniqueRow = idx
			c.applyQualityProps(st, 1, idx)
			c.staffMods(st)

			return true
		}

		tried[idx] = true

		left := false

		for i := 0; i < n; i++ {
			if !tried[i] {
				left = true

				break
			}
		}

		if !left {
			return false
		}
	}
}

// applyQualityProps is the call of ITEMMODS_ApplyPropertyGroup (662420) with
// the properties of a quality row; it belongs to slice C.
func (c *Creator) applyQualityProps(st *itemState, kind, row int) {}

// ---------------------------------------------------------------------------
// low quality (5c0b00 LoD, 5c0890 classic)

// scale75 is the low quality scaling of a stat: 75%, never below min.
func (st *itemState) scale75(stat, min int) {
	v := st.stat(stat) * 75 / 100
	if v < min {
		v = min
	}

	st.setStat(stat, v)
}

func (c *Creator) applyLow(st *itemState) bool {
	lod := st.req.Version >= 1

	b := st.base

	idx := 0
	if c.Quality.Low > 0 {
		idx = st.itemRoll(c.Quality.Low)
	}

	st.uniqueRow = idx

	if st.durable() {
		var d int

		if lod {
			d = b.Durability * 33 / 100
		} else {
			d = b.Durability / 3
		}

		if d < 1 {
			d = 1
		}

		half := d >> 1

		cur := st.unitRoll(half) + half
		if cur == 0 {
			cur = 1
		}

		st.setStat(statDurCur, cur)
		st.setStat(statDurMax, d)
	}

	ok := false

	switch {
	case c.isA(st, "weap"):
		st.scale75(statMaxDam, 2)
		st.scale75(statMinDam, 1)
		st.scale75(statSecMaxDam, 2)
		st.scale75(statSecMinDam, 1)

		if c.itype(st).Throwable {
			st.scale75(statThrowMin, 2)
			st.scale75(statThrowMax, 1)
		}

		ok = true
	case c.isA(st, "armo"):
		st.scale75(statDefense, 1)

		ok = true
	}

	if lod {
		c.staffMods(st)
	}

	return ok
}

// ---------------------------------------------------------------------------
// ethereal (554d90, 660a40)

func (c *Creator) rollEthereal(st *itemState) {
	if st.req.Flags&FlagNoEthereal != 0 {
		return
	}

	if !c.isA(st, "weap") && !c.isA(st, "armo") {
		return
	}

	if !st.durable() || st.quality == QualityLow || st.quality == QualitySet || st.base.Quest != 0 {
		return
	}

	eth := st.itemRoll(100) < 5
	if !eth && st.req.Flags&FlagForceEthereal == 0 {
		return
	}

	st.flags |= flagEthereal

	if c.isA(st, "weap") {
		for _, s := range []int{statMinDam, statMaxDam, statSecMinDam, statSecMaxDam, statThrowMin, statThrowMax} {
			st.setStat(s, st.stat(s)*3/2)
		}
	} else {
		st.setStat(statDefense, st.stat(statDefense)*3/2)
	}

	if st.durable() {
		max := st.stat(statDurMax)/2 + 1

		st.setStat(statDurMax, max)
		st.setStat(statDurCur, max)
	}
}

// ---------------------------------------------------------------------------
// sockets (554c60, 62be00)

// maxSockets is 62bd70: the base item's sockets limited by its type and the
// item level.
func (c *Creator) maxSockets(st *itemState) int {
	ty := c.itype(st)

	limit := ty.MaxSock40

	switch {
	case st.ilvl <= 25:
		limit = ty.MaxSock1
	case st.ilvl <= 40:
		limit = ty.MaxSock25
	}

	if st.base.GemSockets < limit {
		return st.base.GemSockets
	}

	return limit
}

func (c *Creator) rollSockets(st *itemState) {
	b := st.base

	if st.quality < QualityNormal || !b.HasInv || b.Stackable {
		return
	}

	max := c.maxSockets(st)
	if max == 0 {
		return
	}

	switch st.req.Difficulty {
	case 0:
		if max >= 3 {
			max = 3
		}
	case 1:
		if max >= 4 {
			max = 4
		}
	case 2:
		if max >= 6 {
			max = 6
		}
	default:
		if max <= 0 {
			return
		}
	}

	if st.req.Version < 100 && c.isA(st, "tors") {
		return
	}

	roll := st.itemRoll(100)

	if st.req.Flags&FlagNoSockets != 0 {
		return
	}

	if st.req.Flags&FlagForceSockets != 0 {
		roll = 0
	}

	if roll >= 33 {
		return
	}

	st.flags |= flagSocketed

	if st.req.Version >= 1 {
		c.setSockets(st, int(st.itemWord%uint32(max))+1)

		return
	}

	if max > 3 {
		max = 3
	}

	if c.isA(st, "helm") && max > 2 {
		max = 2
	}

	c.setSockets(st, max)
}

// setSockets is 62be00 for the qualities low to superior: the count limited
// by the inventory area, the type and level limit, and at least one.
func (c *Creator) setSockets(st *itemState, n int) {
	b := st.base
	area := b.InvWidth * b.InvHeight

	if area == 0 {
		return
	}

	limit := 6
	if area <= 6 {
		limit = area
	}

	if m := c.maxSockets(st); limit >= m {
		limit = m
	}

	if n < 1 {
		n = 1
	}

	if n >= limit {
		n = limit
	}

	if n <= 0 {
		return
	}

	st.flags |= flagSocketed

	st.setStat(statSockets, n)
}
