package d2drop

import "strings"

// Property functions of classic items (item version 0). ITEMMODS_ApplyProperty
// (662360) sends items of version 0 through a per-property table (741e80:
// handler and one argument, usually the stat id) instead of the function ids
// of Properties.txt. The handlers are the Diablo II 1.09 ones: the maximum of
// a range is exclusive (rand(max-min)+min), class skills use the 1.09 class
// order, and several properties roll a value for each stat they write.
//
// The table is indexed by the Properties.txt row; it is keyed here by the
// property code. Only the 244 rows of the 1.09 table exist: later rows
// (Lord of Destruction properties) find the neighbouring table's entries in
// the game, which nothing but garbage creates; they do nothing here.

type classicHandler int

const (
	chNone       classicHandler = iota
	chStat                      // 65f6f0: roll, write the argument stat
	chStatNamed                 // 65f7e0: the same after ITEM_SetNamedMod
	chDmgPct                    // 65fe80: maximum and minimum damage percent, two rolls
	chRolled                    // 6609c0: roll, write the argument stat
	chClassSkill                // 660840: class skills
	chResAll                    // 65f840: the four resistances, four rolls
	chResAllMax                 // 65f9e0: the four maximum resistances, four rolls
	chSkill                     // 6606e0: skill levels of one skill
	chDmgMin                    // 65fb80
	chDmgMax                    // 65fd00
	chPerLevel                  // 65f620: the stat gets the parameter
	chElem                      // 660380: fixed minimum and maximum damage
	chElemLen                   // 6605d0: fixed minimum, maximum and length
	chAllSkills                 // 6600c0
	chFireSkill                 // 6607a0
	chNoop                      // 6609b0: properties classic items ignore
	chSockets                   // 65f750
	chIndestruct                // 65f7a0
	chPerTime                   // 6602b0
	chCharged                   // 6600f0
)

type classicEntry struct {
	h   classicHandler
	arg int
}

// classicProps is the table at 741e80 by property code.
var classicProps = map[string]classicEntry{
	"ac":              {chStat, 31},
	"ac-miss":         {chStat, 32},
	"ac-hth":          {chStat, 33},
	"red-dmg":         {chStat, 34},
	"red-dmg%":        {chStat, 36},
	"ac%":             {chStatNamed, 16},
	"red-mag":         {chStat, 35},
	"str":             {chStat, 0},
	"dex":             {chStat, 2},
	"vit":             {chStat, 3},
	"enr":             {chStat, 1},
	"mana":            {chStat, 9},
	"mana%":           {chStatNamed, 77},
	"hp":              {chStat, 7},
	"hp%":             {chStatNamed, 76},
	"att":             {chStat, 19},
	"block":           {chStat, 20},
	"cold-min":        {chStat, 54},
	"cold-max":        {chStat, 55},
	"cold-len":        {chStat, 56},
	"fire-min":        {chStat, 48},
	"fire-max":        {chStat, 49},
	"ltng-min":        {chStat, 50},
	"ltng-max":        {chStat, 51},
	"pois-min":        {chStat, 57},
	"pois-max":        {chStat, 58},
	"pois-len":        {chStat, 59},
	"dmg-min":         {chDmgMin, 21},
	"dmg-max":         {chDmgMax, 22},
	"dmg%":            {chDmgPct, -1},
	"dmg-to-mana":     {chStatNamed, 114},
	"res-fire":        {chStat, 39},
	"res-fire-max":    {chStat, 40},
	"res-ltng":        {chStat, 41},
	"res-ltng-max":    {chStat, 42},
	"res-cold":        {chStat, 43},
	"res-cold-max":    {chStat, 44},
	"res-mag":         {chStat, 37},
	"res-mag-max":     {chStat, 38},
	"res-pois":        {chStat, 45},
	"res-pois-max":    {chStat, 46},
	"res-all":         {chResAll, -1},
	"res-all-max":     {chResAllMax, -1},
	"abs-fire%":       {chStat, 142},
	"abs-fire":        {chStat, 143},
	"abs-ltng%":       {chStat, 144},
	"abs-ltng":        {chStat, 145},
	"abs-mag%":        {chStat, 146},
	"abs-mag":         {chStat, 147},
	"abs-cold%":       {chStat, 148},
	"abs-cold":        {chStat, 149},
	"dur":             {chStat, 73},
	"dur%":            {chStatNamed, 75},
	"regen":           {chStat, 74},
	"thorns":          {chStat, 78},
	"swing1":          {chRolled, 93},
	"swing2":          {chRolled, 93},
	"swing3":          {chRolled, 93},
	"gold%":           {chStat, 79},
	"mag%":            {chStat, 80},
	"knock":           {chStat, 81},
	"regen-stam":      {chStatNamed, 28},
	"regen-mana":      {chStatNamed, 27},
	"stam":            {chStat, 11},
	"time":            {chStat, 82},
	"manasteal":       {chStat, 62},
	"lifesteal":       {chStat, 60},
	"ama":             {chClassSkill, 83},
	"pal":             {chClassSkill, 84},
	"nec":             {chClassSkill, 85},
	"sor":             {chClassSkill, 86},
	"bar":             {chClassSkill, 87},
	"herb":            {chStat, 88},
	"light":           {chStat, 89},
	"color":           {chStat, 90},
	"ease":            {chStat, 91},
	"move1":           {chRolled, 96},
	"move2":           {chRolled, 96},
	"move3":           {chRolled, 96},
	"balance1":        {chRolled, 99},
	"balance2":        {chRolled, 99},
	"balance3":        {chRolled, 99},
	"block1":          {chRolled, 102},
	"block2":          {chRolled, 102},
	"block3":          {chRolled, 102},
	"cast1":           {chRolled, 105},
	"cast2":           {chRolled, 105},
	"cast3":           {chRolled, 105},
	"res-pois-len":    {chStat, 110},
	"dmg":             {chStat, 111},
	"howl":            {chStat, 112},
	"stupidity":       {chStat, 113},
	"ignore-ac":       {chStat, 115},
	"reduce-ac":       {chStat, 116},
	"noheal":          {chStat, 117},
	"half-freeze":     {chStat, 118},
	"att%":            {chStatNamed, 119},
	"dmg-ac":          {chStat, 120},
	"dmg-demon":       {chStatNamed, 121},
	"dmg-undead":      {chStatNamed, 122},
	"att-demon":       {chStat, 123},
	"att-undead":      {chStat, 124},
	"throw":           {chStat, 125},
	"fireskill":       {chFireSkill, 126},
	"allskills":       {chAllSkills, 127},
	"light-thorns":    {chStat, 128},
	"freeze":          {chStat, 134},
	"openwounds":      {chStat, 135},
	"crush":           {chStat, 136},
	"kick":            {chStat, 137},
	"mana-kill":       {chStat, 138},
	"demon-heal":      {chStat, 139},
	"bloody":          {chStat, 140},
	"deadly":          {chStat, 141},
	"slow":            {chStat, 150},
	"nofreeze":        {chStat, 153},
	"stamdrain":       {chStat, 154},
	"reanimate":       {chStat, 155},
	"pierce":          {chStat, 156},
	"magicarrow":      {chStat, 157},
	"explosivearrow":  {chStat, 158},
	"dru":             {chClassSkill, 179},
	"ass":             {chClassSkill, 180},
	"skill":           {chSkill, 107},
	"skilltab":        {chNoop, -1},
	"aura":            {chNoop, -1},
	"att-skill":       {chNoop, -1},
	"hit-skill":       {chNoop, -1},
	"gethit-skill":    {chNoop, -1},
	"gembonus":        {chNoop, -1},
	"regen-dur":       {chNoop, -1},
	"fire-fx":         {chNoop, -1},
	"ltng-fx":         {chNoop, -1},
	"sock":            {chSockets, 194},
	"dmg-fire":        {chElem, 48},
	"dmg-ltng":        {chElem, 50},
	"dmg-mag":         {chElem, 52},
	"dmg-cold":        {chElemLen, 54},
	"dmg-pois":        {chElemLen, 57},
	"dmg-throw":       {chElem, 159},
	"dmg-norm":        {chElem, 21},
	"ac/lvl":          {chPerLevel, 214},
	"ac%/lvl":         {chPerLevel, 215},
	"hp/lvl":          {chPerLevel, 216},
	"mana/lvl":        {chPerLevel, 217},
	"dmg/lvl":         {chPerLevel, 218},
	"dmg%/lvl":        {chPerLevel, 219},
	"str/lvl":         {chPerLevel, 220},
	"dex/lvl":         {chPerLevel, 221},
	"enr/lvl":         {chPerLevel, 222},
	"vit/lvl":         {chPerLevel, 223},
	"att/lvl":         {chPerLevel, 224},
	"att%/lvl":        {chPerLevel, 225},
	"dmg-cold/lvl":    {chPerLevel, 226},
	"dmg-fire/lvl":    {chPerLevel, 227},
	"dmg-ltng/lvl":    {chPerLevel, 228},
	"dmg-pois/lvl":    {chPerLevel, 229},
	"res-cold/lvl":    {chPerLevel, 230},
	"res-fire/lvl":    {chPerLevel, 231},
	"res-ltng/lvl":    {chPerLevel, 232},
	"res-pois/lvl":    {chPerLevel, 233},
	"abs-cold/lvl":    {chPerLevel, 234},
	"abs-fire/lvl":    {chPerLevel, 235},
	"abs-ltng/lvl":    {chPerLevel, 236},
	"abs-pois/lvl":    {chPerLevel, 237},
	"thorns/lvl":      {chPerLevel, 238},
	"gold%/lvl":       {chPerLevel, 239},
	"mag%/lvl":        {chPerLevel, 240},
	"regen-stam/lvl":  {chPerLevel, 241},
	"stam/lvl":        {chPerLevel, 242},
	"dmg-dem/lvl":     {chPerLevel, 243},
	"dmg-und/lvl":     {chPerLevel, 244},
	"att-dem/lvl":     {chPerLevel, 245},
	"att-und/lvl":     {chPerLevel, 246},
	"crush/lvl":       {chPerLevel, 247},
	"wounds/lvl":      {chPerLevel, 248},
	"kick/lvl":        {chPerLevel, 249},
	"deadly/lvl":      {chPerLevel, 250},
	"gems%/lvl":       {chNoop, -1},
	"rep-dur":         {chPerLevel, 252},
	"rep-quant":       {chPerLevel, 253},
	"stack":           {chStat, 254},
	"item%":           {chNoop, -1},
	"dmg-slash":       {chNoop, -1},
	"dmg-slash%":      {chNoop, -1},
	"dmg-crush":       {chNoop, -1},
	"dmg-crush%":      {chNoop, -1},
	"dmg-thrust":      {chNoop, -1},
	"dmg-thrust%":     {chNoop, -1},
	"abs-slash":       {chNoop, -1},
	"abs-crush":       {chNoop, -1},
	"abs-thrust":      {chNoop, -1},
	"abs-slash%":      {chNoop, -1},
	"abs-crush%":      {chNoop, -1},
	"abs-thrust%":     {chNoop, -1},
	"ac/time":         {chPerTime, 268},
	"ac%/time":        {chPerTime, 269},
	"hp/time":         {chPerTime, 270},
	"mana/time":       {chPerTime, 271},
	"dmg/time":        {chPerTime, 272},
	"dmg%/time":       {chPerTime, 273},
	"str/time":        {chPerTime, 274},
	"dex/time":        {chPerTime, 275},
	"enr/time":        {chPerTime, 276},
	"vit/time":        {chPerTime, 277},
	"att/time":        {chPerTime, 278},
	"att%/time":       {chPerTime, 279},
	"dmg-cold/time":   {chPerTime, 280},
	"dmg-fire/time":   {chPerTime, 281},
	"dmg-ltng/time":   {chPerTime, 282},
	"dmg-pois/time":   {chPerTime, 283},
	"res-cold/time":   {chPerTime, 284},
	"res-fire/time":   {chPerTime, 285},
	"res-ltng/time":   {chPerTime, 286},
	"res-pois/time":   {chPerTime, 287},
	"abs-cold/time":   {chPerTime, 288},
	"abs-fire/time":   {chPerTime, 289},
	"abs-ltng/time":   {chPerTime, 290},
	"abs-pois/time":   {chPerTime, 291},
	"gold%/time":      {chPerTime, 292},
	"mag%/time":       {chPerTime, 293},
	"regen-stam/time": {chPerTime, 294},
	"stam/time":       {chPerTime, 295},
	"dmg-dem/time":    {chPerTime, 296},
	"dmg-und/time":    {chPerTime, 297},
	"att-dem/time":    {chPerTime, 298},
	"att-und/time":    {chPerTime, 299},
	"crush/time":      {chPerTime, 300},
	"wounds/time":     {chPerTime, 301},
	"kick/time":       {chPerTime, 302},
	"deadly/time":     {chPerTime, 303},
	"gems%/time":      {chNoop, -1},
	"pierce-fire":     {chNoop, -1},
	"pierce-ltng":     {chNoop, -1},
	"pierce-cold":     {chNoop, -1},
	"pierce-pois":     {chNoop, -1},
	"dmg-mon":         {chNoop, -1},
	"dmg%-mon":        {chNoop, -1},
	"att-mon":         {chNoop, -1},
	"att%-mon":        {chNoop, -1},
	"ac-mon":          {chNoop, -1},
	"ac%-mon":         {chNoop, -1},
	"indestruct":      {chIndestruct, -1},
	"charged":         {chCharged, 204},
}

// classicClassOrder is the class number the 1.09 class skills handler stores
// in the parameter, by argument stat (the order is amazon, paladin,
// necromancer, sorceress, barbarian, druid, assassin).
var classicClassOrder = map[int]int{0x53: 0, 0x54: 1, 0x55: 2, 0x56: 3, 0x57: 4, 0xb3: 5, 0xb4: 6}

// classicShift is the stat scaling of 65f350: life, mana and stamina (stats
// 6 to 11) and the per level life and mana (0xd8, 0xd9) are stored in 1/256.
func classicShift(stat, v int) int {
	if (stat >= 6 && stat <= 11) || stat == 0xd8 || stat == 0xd9 {
		return i32(v << 8)
	}

	return v
}

// classicRoll rolls min..max with a maximum that is not included (the item
// version 0 form of the roll: the span is not incremented).
func (pc *propCtx) classicRoll(inst *PropInst) int {
	lo, hi := inst.Min, inst.Max
	if lo == hi {
		return lo
	}

	if lo > hi {
		lo, hi = hi, lo
	}

	return lo + int(pc.st.item.Roll(int32(hi-lo)))
}

// classicStat is 65f450: roll, optionally make sure the base stat exists
// (ITEM_SetNamedMod), write the value to the argument stat.
func (pc *propCtx) classicStat(inst *PropInst, stat int, named bool) int {
	v := pc.classicRoll(inst)

	if named {
		pc.setNamedMod(stat)
	}

	if v == 0 {
		return 0
	}

	pc.write('M', stat, classicShift(stat, v), 0)

	if stat == statSocketFlag {
		pc.write('M', statSocketFlag2, 1, 0)
	}

	return 1
}

// classicValue is 65f580: write a given value to a stat, after the named mod
// call for QualityItems (kind 1) and when asked.
func (pc *propCtx) classicValue(stat, v int, named bool) int {
	if named || pc.kind == PropKindQuality {
		pc.setNamedMod(stat)
	}

	if v == 0 {
		return 0
	}

	pc.write('M', stat, classicShift(stat, v), 0)

	return 1
}

// applyClassic is ITEMMODS_ApplyProperty for items of version 0.
func (pc *propCtx) applyClassic(inst *PropInst) {
	t := pc.c.Props
	if inst.Prop < 0 || inst.Prop >= len(t.Props) {
		return
	}

	e, ok := classicProps[strings.ToLower(t.Props[inst.Prop].Code)]
	if !ok {
		return
	}

	arg := e.arg

	switch e.h {
	case chStat:
		pc.classicStat(inst, arg, false)
	case chStatNamed:
		pc.classicStat(inst, arg, true)
	case chDmgPct:
		pc.classicStat(inst, statMaxDamPct, true)
		pc.classicStat(inst, statMinDamPct, true)
	case chRolled:
		pc.classicValue(arg, pc.classicRoll(inst), false)
	case chResAll:
		for _, s := range []int{0x27, 0x29, 0x2b, 0x2d} {
			pc.classicStat(inst, s, false)
		}
	case chResAllMax:
		for _, s := range []int{0x28, 0x2a, 0x2c, 0x2e} {
			pc.classicStat(inst, s, false)
		}
	case chAllSkills:
		pc.classicStat(inst, arg, false)
	case chClassSkill:
		if v := pc.classicRoll(inst); v != 0 {
			pc.write('M', 0x53, v, classicClassOrder[arg])
		}
	case chSkill:
		v := pc.classicRoll(inst)
		if inst.Param >= 0 && inst.Param < len(t.Skills) {
			pc.write('M', 0x6b, v, inst.Param)
		}
	case chFireSkill:
		if v := pc.classicRoll(inst); v != 0 {
			pc.write('M', 0x7e, v, 1)
		}
	case chDmgMin, chDmgMax:
		pc.classicDamage(inst, e.h == chDmgMin)
	case chPerLevel:
		if inst.Param != 0 {
			pc.classicValue(arg, inst.Param, arg >= 0x10 && arg <= 0x12)
		}
	case chElem:
		pc.classicElem(inst, arg, false)
	case chElemLen:
		pc.classicElem(inst, arg, true)
	case chSockets:
		pc.st.flags |= flagSocketed
		pc.classicSockets(inst.Param)
	case chIndestruct:
		pc.st.setStat(statDurMax, 0)
		pc.st.setStat(statDurCur, 0)
	case chPerTime:
		pc.classicPerTime(inst, arg)
	case chCharged:
		pc.charged(inst, arg)
	case chNoop, chNone:
	}

}

// classicDamage is dmg-min (65fb80) and dmg-max (65fd00): a rolled value for
// the one-handed, two-handed and thrown stats, no floor on the total.
func (pc *propCtx) classicDamage(inst *PropInst, minimum bool) {
	b := pc.st.base
	v := pc.classicRoll(inst)
	weapon := pc.isWeapon()
	oneHand, twoHand := b.MinDam, b.TwoHandMinDam
	st1, st2, st3 := statMinDam, statSecMinDam, statThrowMin

	if !minimum {
		oneHand, twoHand = b.MaxDam, b.TwoHandMaxDam
		st1, st2, st3 = statMaxDam, statSecMaxDam, statThrowMax
	}

	if !(weapon && oneHand == 0 && twoHand != 0) {
		pc.classicValue(st1, v, false)
	}

	if !(weapon && twoHand == 0 && oneHand != 0) {
		pc.classicValue(st2, v, false)
	}

	if !weapon || pc.isThrowable() {
		pc.classicValue(st3, v, false)
	}
}

// classicElem is dmg-fire, dmg-ltng, dmg-mag, dmg-throw, dmg-norm (660380) and
// dmg-cold, dmg-pois (6605d0, with a length): the minimum and maximum are
// taken as they are, no roll. Normal damage goes through the weapon damage
// stats.
func (pc *propCtx) classicElem(inst *PropInst, stat int, withLength bool) {
	b := pc.st.base
	lo, hi := classicShift(stat, inst.Min), classicShift(stat+1, inst.Max)

	if !withLength && stat == statMinDam {
		weapon := pc.isWeapon()

		if !(weapon && b.MaxDam == 0 && b.TwoHandMaxDam != 0) {
			pc.classicValue(statMinDam, lo, false)
			pc.classicValue(statMaxDam, hi, false)
		}

		if !(weapon && b.TwoHandMaxDam == 0 && b.MinDam != 0) {
			pc.classicValue(statSecMinDam, lo, false)
			pc.classicValue(statSecMaxDam, hi, false)
		}

		if pc.isThrowable() {
			pc.classicValue(statThrowMin, lo, false)
			pc.classicValue(statThrowMax, hi, false)
		}

		return
	}

	pc.write('M', stat, lo, 0)
	pc.write('M', stat+1, hi, 0)

	if withLength {
		pc.write('M', stat+2, classicShift(stat+2, inst.Param), 0)
	}

	if stat == 0x39 {
		pc.write('M', statSocketFlag2, 1, 0)
	}
}

// classicSockets is ITEM_SetSockets (62bf50): the number of sockets, at least
// 1, at most the grid of the item, 6 and what the item type allows.
func (pc *propCtx) classicSockets(n int) {
	b := pc.st.base

	limit := b.InvWidth * b.InvHeight
	if limit <= 0 {
		return
	}

	if limit > 6 {
		limit = 6
	}

	if m := pc.maxSockets(); limit >= m {
		limit = m
	}

	n = minInt(maxInt(n, 1), limit)

	pc.st.flags |= flagSocketed
	pc.st.setStat(statSockets, n)
}

// classicPerTime is the /time properties of 6602b0: the period (param, 0-3),
// the minimum and the maximum (+256, 10 bits each) packed in one stat value.
func (pc *propCtx) classicPerTime(inst *PropInst, stat int) {
	lo, hi := inst.Min+0x100, inst.Max+0x100

	if inst.Param < 0 || inst.Param > 3 || lo < 0 || lo > 0x3ff || hi < 0 || hi > 0x3ff {
		return // the game reports an error here
	}

	pc.write('L', stat, inst.Param+(hi<<10+lo)*4, 0)
}
