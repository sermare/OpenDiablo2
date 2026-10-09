package d2drop

// The property engine of Game.exe 1.14b: ITEMMODS_ApplyPropertyGroup (662420)
// applies the properties of an affix, a QualityItems row, a unique or a set
// item to the item. Each property (a row of Properties.txt) has up to seven
// slots; each slot names a property function (table at 742620, ids 1-36) that
// turns the {property, param, min, max} instance into stat writes on the
// item. Everything here is verified against the emulated game
// (testdata/create_c.json): the stat writes, their order and the item
// generator after each group.
//
// Items of Lord of Destruction version (Request.Version >= 1) use the
// property functions below; classic items (version 0) go through the per
// property table of props_classic.go.

// PropInst is one property instance of a record: the row of Properties.txt
// (Prop, -1 for none), its parameter and its minimum and maximum.
type PropInst struct {
	Prop, Param, Min, Max int
}

// Property kinds of ITEMMODS_ApplyPropertyGroup.
const (
	PropKindAffix   = 0 // magic / rare affix: 3 properties
	PropKindQuality = 1 // QualityItems: 2 properties
	PropKindGem     = 2 // gems: 3 properties of one of three blocks
	PropKindUnique  = 3 // UniqueItems: 12 properties
	PropKindSetItem = 4 // SetItems: 9 properties and the set bonus properties
	PropKindRune    = 5 // like 2
	PropKindSocket6 = 6 // single property of a socketed item (record is the instance)
	PropKindSocket7 = 7
)

// PropRow is a row of Properties.txt.
type PropRow struct {
	Code string
	// Set, Val, Func and Stat are the seven slots: the set column (non zero:
	// write with the list "add" call), the val column, the property function
	// id (0 ends the slots) and the stat id (ItemStatCost id).
	Set, Val, Func, Stat [7]int
}

// SkillInfo is the part of a Skills.txt row the property functions read.
type SkillInfo struct {
	ReqLevel int // reqlevel
	MaxLevel int // maxlvl (0 = none: the game uses 20)
}

// PropTables holds Properties.txt, the valshift column of ItemStatCost.txt
// and the skill levels.
type PropTables struct {
	Props  []PropRow
	ByCode map[string]int
	// ValShift is ItemStatCost's ValShift by stat id.
	ValShift []int
	Skills   []SkillInfo
	// SkillParamShift and SkillParamMask encode a skill and its level in the
	// param of the charged skill stat: (skill << shift) + (level & mask).
	// The game reads them from its stat tables (6 and 0x3f).
	SkillParamShift, SkillParamMask int
	// Affix holds the properties of the rows of the combined affix table
	// (magic suffixes, magic prefixes, automagic; ids from 1) and Quality
	// those of the rows of QualityItems.txt.
	Affix   [][3]PropInst
	Quality [][2]PropInst
}

// Stats and flags only the property engine uses (the others are slice A's).
const (
	statMinDamPct   = 0x12
	statMaxDamPct   = 0x11
	statMaxDurPct   = 0x4b
	statSocketFlag  = 0x3a
	statSocketFlag2 = 0x146

	maxPropFunc = 37 // number of slots of the function table (742620)
)

// propCtx is one property application.
type propCtx struct {
	c    *Creator
	st   *itemState
	kind int
	// sel is the selector of the item stat list the stats go to (0: the base
	// list, 0xa5..0xa9: set bonus tiers).
	sel int
}

// propArgs are the arguments the dispatcher (6622c0) hands a property
// function.
type propArgs struct {
	inst *PropInst
	set  int // set column of the slot
	stat int // stat id of the slot
	val  int // val column of the slot
	prev int // result of the first slot's function
}

type propFunc func(pc *propCtx, a propArgs) int

var propFuncs [maxPropFunc]propFunc

func init() {
	propFuncs[1] = pfStatRolled(false, true)
	propFuncs[2] = pfStatRolled(false, false)
	propFuncs[3] = pfStatRolled(true, true)
	propFuncs[4] = pfStatRolled(true, false)
	propFuncs[5] = pfMinDamage
	propFuncs[6] = pfMaxDamage
	propFuncs[7] = pfDamagePercent
	propFuncs[8] = pfPlain
	propFuncs[9] = pfSkillParam
	propFuncs[10] = pfSkillTab
	propFuncs[11] = pfProcSkill
	propFuncs[12] = pfParamInst
	propFuncs[13] = pfDurabilityPercent
	propFuncs[14] = pfSockets
	propFuncs[15] = pfElemMin
	propFuncs[16] = pfElemMax
	propFuncs[17] = pfParamValue
	propFuncs[18] = pfPerTime
	propFuncs[19] = pfCharged
	propFuncs[20] = pfIndestructible
	propFuncs[21] = pfClassSkills
	propFuncs[22] = pfSkillLevel
	propFuncs[23] = pfEthereal
	propFuncs[24] = pfPlainParam
	propFuncs[36] = pfRandClassSkill
}

func i32(x int) int { return int(int32(x)) }

// applyProps applies the properties of one record to the item
// (ITEMMODS_ApplyPropertyGroup, 662420). For the kinds 0, 1, 2 and 5 props is
// the record's list (3, 2, 3 and 3 entries); for the kinds 3 and 4 it is
// ignored: the unique or set item row is the one in st.uniqueRow.
func (c *Creator) applyProps(st *itemState, kind int, props []PropInst) {
	pc := &propCtx{c: c, st: st, kind: kind}

	switch kind {
	case PropKindUnique:
		u := c.uniqueRow(st.uniqueRow)
		if u == nil {
			return
		}

		for i := range u.Props {
			pc.applyInst(&u.Props[i])
		}
	case PropKindSetItem:
		c.applySetItemProps(pc, c.setItemRow(st.uniqueRow))
	default:
		for i := range props {
			pc.applyInst(&props[i])
		}
	}
}

// applyAffixProps applies the properties of the affix with the 1-based id of
// the combined affix table (kind 0).
func (c *Creator) applyAffixProps(st *itemState, id int) {
	if c.Props == nil || id < 1 || id > len(c.Props.Affix) {
		return
	}

	c.applyProps(st, PropKindAffix, c.Props.Affix[id-1][:])
}

// applyQualityProps applies the properties of a QualityItems row (kind 1).
func (c *Creator) applyQualityProps(st *itemState, kind, row int) {
	if c.Props == nil || row < 0 || row >= len(c.Props.Quality) {
		return
	}

	c.applyProps(st, kind, c.Props.Quality[row][:])
}

// applyInstance applies one property instance to the stat list with the given
// selector: what ITEMMODS_ApplyPropertyGroup does for each property of the
// kinds 6 and 7 (runeword and socketed gem properties, list 0xab for
// runewords). The property functions are verified for these kinds, the code
// that walks a runeword's seven properties (662620) is not ported.
func (c *Creator) applyInstance(st *itemState, kind int, inst PropInst, sel int) {
	(&propCtx{c: c, st: st, kind: kind, sel: sel}).applyInst(&inst)
}

func (c *Creator) uniqueRow(i int) *UniqueItem {
	if c.Uniques == nil || i < 0 || i >= len(c.Uniques.Uniques) {
		return nil
	}

	return &c.Uniques.Uniques[i]
}

func (c *Creator) setItemRow(i int) *SetItem {
	if c.Uniques == nil || i < 0 || i >= len(c.Uniques.SetItems) {
		return nil
	}

	return &c.Uniques.SetItems[i]
}

// setBonusSelector and setBonusMask are the tables at 6ef0f0 and 6ef10c,
// indexed by tier+2.
var setBonusSelector = [...]int{0, 0, 0xa5, 0xa6, 0xa7, 0xa8, 0xa9}

// applySetItemProps is kind 4: the nine properties of the set item, then the
// ten set bonus properties (aprop1a..5b). With "add func" 0 the bonuses go to
// the item's base list; otherwise each tier has its own list (selectors
// 0xa5..0xa9), which the game activates when enough set items are worn.
func (c *Creator) applySetItemProps(pc *propCtx, s *SetItem) {
	if s == nil {
		return
	}

	// Classic items (version 0) only get two properties.
	n := len(s.Props)
	if pc.st.req.Version == 0 {
		n = 2
	}

	for i := 0; i < n; i++ {
		pc.applyInst(&s.Props[i])
	}

	if pc.st.req.Version == 0 {
		return
	}

	for i := range s.AProps {
		pc.sel = 0
		if s.AddFunc != 0 {
			pc.sel = setBonusSelector[i/2+2]
		}

		pc.applyInst(&s.AProps[i])
	}

	pc.sel = 0
}

// applyInst is ITEMMODS_ApplyProperty (662360 / 6622c0, Lord of Destruction
// path): run the slots of the property one after another, the result of the
// first slot is handed to the later ones.
func (pc *propCtx) applyInst(inst *PropInst) {
	t := pc.c.Props
	if inst == nil || inst.Prop < 0 || t == nil || inst.Prop >= len(t.Props) {
		return
	}

	if pc.st.req.Version < 1 {
		pc.applyClassic(inst)

		return
	}

	row := &t.Props[inst.Prop]
	prev := 0

	for i := 0; i < 7; i++ {
		f := row.Func[i]
		if f >= maxPropFunc || f < 0 || propFuncs[f] == nil {
			break
		}

		r := propFuncs[f](pc, propArgs{inst: inst, set: row.Set[i], stat: row.Stat[i], val: row.Val[i], prev: prev})
		if i == 0 {
			prev = r
		}
	}
}

// roll is ITEMMODS_RollPropertyValue (660f40) on the item generator.
func (pc *propCtx) roll(inst *PropInst) int {
	lo, hi := inst.Min, inst.Max
	if lo == hi {
		return lo
	}

	if lo > hi {
		lo, hi = hi, lo
	}

	return lo + int(pc.st.item.Roll(int32(hi-lo+1)))
}

// statTotal is the value of a stat as the game reads it from the item: the
// unit's own value plus the flat values of the item's stat lists (what the
// properties wrote). Percentages are not applied.
func (st *itemState) statTotal(stat int) int {
	total := st.stat(stat)

	for _, w := range st.writes {
		if w.Kind != 'S' && w.Stat == stat {
			total += w.Value
		}
	}

	return total
}

// unitStat returns the last value set on the item unit's own stat list.
func (st *itemState) unitStat(stat int) (int, bool) {
	for i := len(st.writes) - 1; i >= 0; i-- {
		if w := st.writes[i]; w.Kind == 'S' && w.Stat == stat {
			return w.Value, true
		}
	}

	return 0, false
}

func (pc *propCtx) isWeapon() bool {
	return pc.c.Items != nil && pc.c.Items.IsA(pc.st.base, "weap")
}

func (pc *propCtx) isArmor() bool {
	return pc.c.Items != nil && pc.c.Items.IsA(pc.st.base, "armo")
}

// isThrowable is ITEM_IsThrowableType (62bbd0): the ItemTypes flag of the
// item's type.
func (pc *propCtx) isThrowable() bool {
	ty := pc.c.Items.Types[pc.st.base.Type]

	return ty != nil && ty.Throwable
}

// add is ITEMMODS_AddPropertyStat (660fa0): write value << valshift to the
// item's stat list. A non zero set column uses the list's "add" call ('L'),
// zero the "set" call ('M'). It returns the unshifted value (0 when nothing
// was written).
func (pc *propCtx) add(set, stat, param, value int) int {
	t := pc.c.Props
	if value == 0 {
		return 0
	}

	stat = int(int16(stat))
	if stat < 0 || stat >= len(t.ValShift) {
		return 0
	}

	kind := byte('M')
	if set != 0 {
		kind = 'L'
	}

	pc.write(kind, stat, i32(int(uint32(value)<<uint(t.ValShift[stat]))), param)

	if stat == statSocketFlag {
		if set != 0 {
			if !pc.listHas(statSocketFlag2) {
				pc.write('L', statSocketFlag2, 1, 0)
			}
		} else {
			pc.write('M', statSocketFlag2, 1, 0)
		}
	}

	return value
}

func (pc *propCtx) write(kind byte, stat, value, param int) {
	pc.st.writes = append(pc.st.writes, StatWrite{Kind: kind, Stat: stat, Value: value, Param: param, List: pc.sel})
}

// listHas reports whether the current stat list has a non zero value of the
// stat.
func (pc *propCtx) listHas(stat int) bool {
	has := false

	for _, w := range pc.st.writes {
		if w.Kind != 'S' && w.List == pc.sel && w.Stat == stat {
			has = w.Value != 0
		}
	}

	return has
}

// setNamedMod is ITEM_SetNamedMod (65f1d0): before a property changes a
// damage or defense stat, make sure the unit stat holds the base item's
// value.
func (pc *propCtx) setNamedMod(stat int) {
	st, b := pc.st, pc.st.base

	switch stat {
	case 0x10, 0x1f:
		if pc.isArmor() && b.MaxAC != 0 {
			cur, _ := st.unitStat(statDefense)
			v := cur + 1

			if v <= b.MaxAC {
				v = b.MaxAC + 1
			}

			st.setStat(statDefense, v)
		}
	case 0x11, 0x16:
		if pc.isWeapon() {
			pc.baseMax()
		}
	case 0x12, 0x15:
		if pc.isWeapon() {
			pc.baseMin()
		}
	}
}

// baseMax writes the base weapon's maximum damage stats on the unit.
func (pc *propCtx) baseMax() {
	st, b := pc.st, pc.st.base

	if b.MaxDam != 0 {
		st.setStat(statMaxDam, b.MaxDam)
	}

	if b.TwoHandMaxDam != 0 {
		st.setStat(statSecMaxDam, b.TwoHandMaxDam)
	}

	if pc.isThrowable() && b.MaxMisDam != 0 {
		st.setStat(statThrowMax, b.MaxMisDam)
	}
}

// baseMin writes the base weapon's minimum damage stats on the unit.
func (pc *propCtx) baseMin() {
	st, b := pc.st, pc.st.base

	if b.MinDam != 0 {
		st.setStat(statMinDam, b.MinDam)
	}

	if b.TwoHandMinDam != 0 {
		st.setStat(statSecMinDam, b.TwoHandMinDam)
	}

	if pc.isThrowable() && b.MinMisDam != 0 {
		st.setStat(statThrowMin, b.MinMisDam)
	}
}

// pfStatRolled builds the property functions 1-4: roll min..max (functions 3
// and 4 take the result of the first slot instead when there is one) and add
// it to the stat. Functions 1 and 3 tell setNamedMod only for QualityItems
// (kind 1).
func pfStatRolled(usePrev, onlyQuality bool) propFunc {
	return func(pc *propCtx, a propArgs) int {
		if !onlyQuality || pc.kind == PropKindQuality {
			pc.setNamedMod(a.stat)
		}

		v := a.prev
		if !usePrev || v == 0 {
			v = pc.roll(a.inst)
		}

		return pc.add(a.set, a.stat, 0, v)
	}
}

// damageAdjust keeps the sum of the base damage and the added value above a
// floor: a minimum damage is floored at 1 (the value becomes 1-base), a
// maximum damage at 0 (the value becomes -base).
func damageAdjust(base, v int, minimum bool) int {
	if base <= 0 {
		return v
	}

	if base+v > 0 {
		return v
	}

	if minimum {
		return 1 - base
	}

	return -base
}

// pfMinDamage is function 5 (dmg-min), pfMaxDamage function 6 (dmg-max): add
// to the minimum / maximum damage of the one-handed, two-handed and thrown
// weapon stats, keeping the total damage positive.
func pfMinDamage(pc *propCtx, a propArgs) int { return pc.damage(a, true) }
func pfMaxDamage(pc *propCtx, a propArgs) int { return pc.damage(a, false) }

func (pc *propCtx) damage(a propArgs, minimum bool) int {
	b := pc.st.base
	v := a.prev

	if v == 0 {
		v = pc.roll(a.inst)
	}

	weapon := pc.isWeapon()
	oneHand, twoHand, thrown := b.MinDam, b.TwoHandMinDam, b.MinMisDam
	st1, st2, st3 := statMinDam, statSecMinDam, statThrowMin

	if !minimum {
		oneHand, twoHand, thrown = b.MaxDam, b.TwoHandMaxDam, b.MaxMisDam
		st1, st2, st3 = statMaxDam, statSecMaxDam, statThrowMax
	}

	// One-handed stat: skipped for a weapon that only has a two-handed value.
	if !(weapon && oneHand == 0 && twoHand != 0) {
		pc.add(a.set, st1, 0, damageAdjust(oneHand, v, minimum))
	}

	// Two-handed stat: skipped for a weapon that only has a one-handed value.
	if !(weapon && twoHand == 0 && oneHand != 0) {
		pc.add(a.set, st2, 0, damageAdjust(twoHand, v, minimum))
	}

	// Thrown stat: for weapons only when the type is throwable.
	if !weapon || pc.isThrowable() {
		pc.add(a.set, st3, 0, damageAdjust(thrown, v, minimum))
	}

	return v
}

// pfDamagePercent is function 7 (dmg%).
func pfDamagePercent(pc *propCtx, a propArgs) int {
	v := a.prev
	if v == 0 {
		v = pc.roll(a.inst)
	}

	b := pc.st.base
	percent := func() int {
		pc.add(a.set, statMinDamPct, 0, v)
		pc.add(a.set, statMaxDamPct, 0, v)

		return v
	}

	weapon := pc.isWeapon()

	if weapon {
		pc.baseMin()
		pc.baseMax()
	}

	if !weapon {
		return percent()
	}

	maxBase := b.MaxDam
	if b.TwoHandMaxDam > maxBase {
		maxBase = b.TwoHandMaxDam
	}

	if i32(maxBase*v)/100 > 0 {
		return percent()
	}

	// Too small to move a percentage: a flat +1 maximum damage instead.
	return pc.damage(propArgs{inst: a.inst, set: a.set, stat: statMaxDam, val: a.val, prev: 1}, false)
}

// pfPlain is function 8: the rolled value, no parameter.
func pfPlain(pc *propCtx, a propArgs) int {
	v := a.prev
	if v == 0 {
		v = pc.roll(a.inst)
	}

	return pc.add(a.set, a.stat, 0, v)
}

// pfPlainParam is function 24: like 8 with the instance's parameter.
func pfPlainParam(pc *propCtx, a propArgs) int {
	v := a.prev
	if v == 0 {
		v = pc.roll(a.inst)
	}

	return pc.add(a.set, a.stat, a.inst.Param, v)
}

// pfSkillParam is function 9: the parameter is a skill id.
func pfSkillParam(pc *propCtx, a propArgs) int {
	v := a.prev
	if v == 0 {
		v = pc.roll(a.inst)
		if v == 0 {
			return 0
		}
	}

	if a.inst.Param < 0 || a.inst.Param >= len(pc.c.Props.Skills) {
		return 0
	}

	return pc.add(a.set, a.stat, a.inst.Param, v)
}

// pfSkillTab is function 10: the parameter is a skill tab (3 per class).
func pfSkillTab(pc *propCtx, a propArgs) int {
	v := a.prev
	if v == 0 {
		v = pc.roll(a.inst)
		if v == 0 {
			return 0
		}
	}

	p := a.inst.Param

	return pc.add(a.set, a.stat, p%3+(p/3)*8, v)
}

// pfParamInst is function 12 (skill-rand): the roll is the parameter and the
// instance's parameter is the value (+N levels to a random skill of the
// range).
func pfParamInst(pc *propCtx, a propArgs) int {
	r := pc.roll(a.inst)

	return pc.add(a.set, a.stat, r, a.inst.Param)
}

// pfClassSkills is function 21: roll, the parameter is the val column.
func pfClassSkills(pc *propCtx, a propArgs) int {
	return pc.add(a.set, a.stat, a.val, pc.roll(a.inst))
}

// pfSkillLevel is function 22: roll, the parameter is the instance's
// parameter as an unsigned 16 bit number.
func pfSkillLevel(pc *propCtx, a propArgs) int {
	return pc.add(a.set, a.stat, a.inst.Param&0xffff, pc.roll(a.inst))
}

// pfRandClassSkill is function 36 (randclassskill): the roll is the
// parameter (the class), the val column the value, as a signed 16 bit number.
func pfRandClassSkill(pc *propCtx, a propArgs) int {
	r := pc.roll(a.inst)

	return pc.add(a.set, a.stat, r, int(int16(a.val)))
}

// pfDurabilityPercent is function 13 (dur%): roll and add, then raise the
// current durability to the (now larger) maximum.
func pfDurabilityPercent(pc *propCtx, a propArgs) int {
	if pc.kind == PropKindQuality {
		pc.setNamedMod(a.stat)
	}

	v := pc.add(a.set, a.stat, 0, pc.roll(a.inst))
	if v == 0 {
		return 0
	}

	if max := pc.st.maxDurability(); max > 0 {
		pc.st.setStat(statDurCur, max)
	}

	return v
}

// maxDurability is the maximum durability stat of the item as the game reads
// it (626060, ITEM_GetStat of 0x49): the unit's own value plus the flat
// values of the item's stat lists, raised by the maximum durability
// percentages (stat 0x4b, op 13 of ItemStatCost). Every list counts, the set
// bonus tiers too (UNVERIFIED for real set items: no set bonus has these
// stats).
func (st *itemState) maxDurability() int {
	max, ok := st.unitStat(statDurMax)
	if !ok {
		return 0
	}

	pct := 0

	for _, w := range st.writes {
		if w.Kind == 'S' {
			continue
		}

		switch w.Stat {
		case statDurMax:
			max += w.Value
		case statMaxDurPct:
			pct += w.Value
		}
	}

	return max + max*pct/100
}

// pfSockets is function 14 (sock): number of sockets, limited by the item's
// grid, the item type and the item level.
func pfSockets(pc *propCtx, a propArgs) int {
	b := pc.st.base

	limit := b.InvWidth * b.InvHeight
	if limit <= 0 {
		return 0
	}

	if limit > 6 {
		limit = 6
	}

	if m := pc.maxSockets(); limit >= m {
		limit = m
	}

	v := a.prev
	if v <= 0 {
		v = pc.roll(a.inst)
		if v <= 0 {
			v = a.inst.Param
		}
	}

	switch {
	case maxInt(v, 1) >= limit:
		v = limit
	case v <= 1:
		v = 1
	}

	if v <= 0 {
		return 0
	}

	pc.st.flags |= flagSocketed
	pc.st.setStat(statSockets, v)

	return v
}

// maxSockets is ITEM_GetMaxSockets (62bd70): the base item's gemsockets
// limited by the item type's maximum for the item level.
func (pc *propCtx) maxSockets() int {
	b := pc.st.base
	ty := pc.c.Items.Types[b.Type]

	if ty == nil {
		return 0
	}

	lim := ty.MaxSock40

	switch {
	case pc.st.ilvl <= 25:
		lim = ty.MaxSock1
	case pc.st.ilvl <= 40:
		lim = ty.MaxSock25
	}

	return minInt(b.GemSockets, lim)
}

// pfElemMin is function 15, pfElemMax function 16: the minimum / maximum of
// an elemental damage: the value is the instance's min / max, no roll. A
// damage stat goes through the weapon damage functions.
func pfElemMin(pc *propCtx, a propArgs) int {
	v := a.inst.Min
	if a.stat == statMinDam {
		a.prev = v
		pfMinDamage(pc, a)

		return v
	}

	pc.add(a.set, a.stat, 0, v)

	return v
}

func pfElemMax(pc *propCtx, a propArgs) int {
	v := a.inst.Max
	if a.stat == statMaxDam {
		a.prev = v
		pfMaxDamage(pc, a)

		return v
	}

	pc.add(a.set, a.stat, 0, v)

	return v
}

// pfParamValue is function 17 (/lvl and length properties): the value is the
// instance's parameter, or a roll when it is 0.
func pfParamValue(pc *propCtx, a propArgs) int {
	v := a.inst.Param
	if v == 0 {
		v = pc.roll(a.inst)
		if v == 0 {
			return 0
		}
	}

	if a.stat == statMaxDam {
		a.prev = v
		pfMaxDamage(pc, a)

		return v
	}

	pc.add(a.set, a.stat, 0, v)

	return v
}

// pfPerTime is function 18 (/time properties): the stat packs the period
// (param, 0-3) and the minimum and maximum (+256, 10 bits each) in one value.
func pfPerTime(pc *propCtx, a propArgs) int {
	t := pc.c.Props
	stat := int(int16(a.stat))

	if stat < 0 || stat >= len(t.ValShift) {
		return 0
	}

	clamp := func(x, hi int) int {
		if x <= 0 {
			return 0
		}

		return minInt(x, hi)
	}

	period := minInt(clamp(a.inst.Param, 3), 3)
	lo := clamp(a.inst.Min+0x100, 0x3ff)
	hi := clamp(a.inst.Max+0x100, 0x3ff)

	pc.write('L', stat, period+(hi<<10+lo)*4, 0)

	return hi
}

// pfIndestructible is function 20: the indestructible stat.
func pfIndestructible(pc *propCtx, a propArgs) int {
	if len(pc.c.Props.ValShift) <= statNoDurab {
		return 0
	}

	pc.write('M', statNoDurab, 1, 0)

	return 1
}

// pfEthereal is function 23.
func pfEthereal(pc *propCtx, a propArgs) int {
	st := pc.st
	if st.flags&flagEthereal != 0 || !pc.hasDurability() {
		return 0
	}

	pc.c.makeEthereal(st)

	return 1
}

// hasDurability is ITEM_HasDurability (629b00): the base item has a
// durability, can lose it, and is not indestructible.
func (pc *propCtx) hasDurability() bool {
	b := pc.st.base
	if b.NoDurability || b.Durability == 0 {
		return false
	}

	if _, ok := pc.st.unitStat(statDurMax); !ok {
		return false
	}

	return pc.st.statTotal(statNoDurab) <= 0
}

// skillLevel is the level a charged skill or a skill proc gets from the item
// level when the instance gives none: (ilvl - reqlevel) / 4 + 1, at most the
// skill's maximum; a negative maximum is a divisor of the span up to level 99.
func (pc *propCtx) skillLevel(inst *PropInst) int {
	sk := pc.c.Props.Skills[inst.Param]
	req := sk.ReqLevel
	mx := sk.MaxLevel

	if mx <= 0 {
		mx = 20
	}

	ilvl := pc.st.ilvl

	switch {
	case inst.Max == 0:
		lvl := maxInt((ilvl-req)/4+1, 1)
		if lvl >= mx {
			return mx
		}

		return lvl
	case inst.Max < 0:
		span := maxInt(99-req, 1)
		div := maxInt(-(span / inst.Max), 1)
		lvl := (ilvl - req) / div

		if lvl <= 0 {
			lvl = 1
		}

		return lvl
	}

	return inst.Max
}

// pfProcSkill is function 11 (att-skill, hit-skill, gethit-skill, kill-skill,
// death-skill, levelup-skill): the value is the chance (min, 5 when not
// positive), the parameter encodes the skill and its level.
func pfProcSkill(pc *propCtx, a propArgs) int {
	t := pc.c.Props
	if a.inst.Param < 0 || a.inst.Param >= len(t.Skills) {
		return 0
	}

	chance := a.inst.Min
	if chance <= 0 {
		chance = 5
	}

	lvl := pc.skillLevel(a.inst) & 0x3f

	return pc.add(a.set, a.stat, a.inst.Param<<6+lvl, chance)
}

// pfCharged is function 19 (charged): a skill with charges.
func pfCharged(pc *propCtx, a propArgs) int {
	return pc.charged(a.inst, a.stat)
}

// charged writes a skill with charges to the stat. The stat value holds the
// maximum charges in bits 8 and up and the current charges in the low byte;
// the current charges are rolled on the item generator.
func (pc *propCtx) charged(inst *PropInst, statID int) int {
	t := pc.c.Props
	if inst.Param < 0 || inst.Param >= len(t.Skills) {
		return 0
	}

	stat := int(int16(statID))
	if stat < 0 || stat >= len(t.ValShift) {
		return 0
	}

	lvl := pc.skillLevel(inst)

	charges := inst.Min

	switch {
	case charges == 0:
		charges = 5
	case charges < 0:
		charges = -charges
		charges += charges * lvl / 8
	}

	switch {
	case charges <= 1:
		charges = 1
	case charges >= 0xff:
		charges = 0xff
	}

	low := charges / 8
	cur := int(pc.st.item.Roll(int32(charges-low))) + low + 1

	param := inst.Param<<uint(t.SkillParamShift) + (lvl & t.SkillParamMask)
	pc.write('L', stat, cur&0xff+charges<<8, param)

	return charges
}
