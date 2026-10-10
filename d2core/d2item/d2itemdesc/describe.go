package d2itemdesc

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2fileformats/d2s"
)

// Color is a tooltip line color; the game's item colors by quality.
type Color uint8

// Tooltip colors.
const (
	White  Color = iota // normal items, base stats
	Blue                // magic properties and modified base stats
	Yellow              // rare items
	Gold                // unique items, set name
	Green               // set items and active set bonuses
	Orange              // crafted items and runes
	Gray                // low quality, ethereal and socketed normal items
	Red                 // unmet requirements, unidentified, missing set pieces
)

func (c Color) String() string {
	return [...]string{"white", "blue", "yellow", "gold", "green", "orange", "gray", "red"}[c]
}

// Line is one coloured tooltip line.
type Line struct {
	Text  string
	Color Color
}

// Context is what the description needs to know about the player. The zero
// value (or nil) describes the item as if no hero were looking at it: no
// requirement is shown as unmet, no set pieces are worn besides the item.
type Context struct {
	HasHero   bool
	CharLevel int
	Strength  int
	Dexterity int
	Class     int // charstats row (0 Amazon .. 6 Assassin), -1 unknown

	// WornSetItems are the setitems.txt index names of the set pieces the
	// hero wears (the described item included, when it is worn).
	WornSetItems map[string]bool
}

// Class codes of ItemTypes.txt's Class column.
var classCodes = map[string]int{"ama": 0, "sor": 1, "nec": 2, "pal": 3, "bar": 4, "dru": 5, "ass": 6}

// Format substitutes %0 %1 %2 .. in a string table format ("%0 %1 %2" of
// MagicFormat) and drops the double spaces left by missing parts.
func Format(format string, parts ...string) string {
	out := format

	for i := len(parts) - 1; i >= 0; i-- {
		out = strings.ReplaceAll(out, "%"+strconv.Itoa(i), parts[i])
	}

	return strings.Join(strings.Fields(out), " ")
}

func (t *Tables) format2(key, def string, parts ...string) string {
	f := t.Tr(key)
	if f == "" {
		f = def
	}

	return Format(f, parts...)
}

// Describe returns the tooltip of a saved item, line by line with colors.
// ctx may be nil.
func (t *Tables) Describe(it *d2s.Item, ctx *Context) []Line {
	if it.Ear {
		return t.describeEar(it)
	}

	base := t.Bases[it.Code]
	if base == nil {
		return []Line{{it.Code, White}}
	}

	d := &desc{t: t, it: it, base: base, ctx: ctx}

	return d.build()
}

func (t *Tables) describeEar(it *d2s.Item) []Line {
	if it.EarInfo == nil {
		return []Line{{"Ear", White}}
	}

	return []Line{
		{fmt.Sprintf("%s (%s %d)", it.EarInfo.Name, t.tr("Level"), it.EarInfo.Level), Gold},
	}
}

type desc struct {
	t    *Tables
	it   *d2s.Item
	base *BaseItem
	ctx  *Context

	stats    []Stat // merged: own + runeword + sockets
	kindName string
}

// shieldType etc: socket effects depend on the kind of item they sit in.
func (d *desc) socketKind() int {
	switch {
	case d.base.Weapon:
		return 0
	case d.t.IsA(d.base.Type, "shld"):
		return 2
	}

	return 1
}

func toStats(props []d2s.Property) []Stat {
	out := make([]Stat, 0, len(props))
	for _, p := range props {
		out = append(out, Stat{ID: p.ID, Param: int(p.Param), Value: int(p.Value)})
	}

	return out
}

// SocketedStats returns the stats a child item (gem, rune, jewel) gives to
// the item it is socketed in. Runes and gems have no stats of their own in a
// save; their effect comes from gems.txt by the kind of the parent.
func (t *Tables) SocketedStats(child *d2s.Item, parent *BaseItem) []Stat {
	if len(child.Properties) > 0 { // jewels carry their magic properties
		return toStats(child.Properties)
	}

	g := t.Gems[child.Code]
	if g == nil {
		return nil
	}

	var specs []PropSpec

	switch {
	case parent.Weapon:
		specs = g.Weapon
	case t.IsA(parent.Type, "shld"):
		specs = g.Shield
	default:
		specs = g.Helm
	}

	var out []Stat
	for _, s := range specs {
		out = append(out, t.EvalProp(s)...)
	}

	return out
}

func (d *desc) collect() {
	lists := [][]Stat{toStats(d.it.Properties), toStats(d.it.RunewordProperties)}

	for i := range d.it.Children {
		lists = append(lists, d.t.SocketedStats(&d.it.Children[i], d.base))
	}

	d.stats = Merge(lists...)
}

func (d *desc) value(id int) int {
	for _, s := range d.stats {
		if s.ID == id && s.Param == 0 {
			return s.Value
		}
	}

	return 0
}

func (d *desc) has(id int) bool {
	for _, s := range d.stats {
		if s.ID == id {
			return true
		}
	}

	return false
}

func (d *desc) identified() bool { return d.it.Identified || d.it.Simple }

// build assembles the tooltip in the order of the original game: names,
// base statistics, requirements, magical properties, set information,
// then the footnotes.
func (d *desc) build() []Line {
	d.collect()

	lines := d.nameLines()
	lines = append(lines, d.socketableLines()...)
	lines = append(lines, d.potionLines()...)
	lines = append(lines, d.baseLines()...)
	lines = append(lines, d.propertyLines()...)
	lines = append(lines, d.footLines()...)
	lines = append(lines, d.setLines()...)

	return lines
}

func (d *desc) qualityColor() Color {
	it := d.it

	switch it.Quality {
	case d2s.QualityMagic:
		return Blue
	case d2s.QualityRare:
		return Yellow
	case d2s.QualityCrafted:
		return Orange
	case d2s.QualitySet:
		return Green
	case d2s.QualityUnique:
		return Gold
	case d2s.QualityLow:
		return Gray
	}

	if d.isRune() {
		return Orange
	}

	if it.Runeword || it.Socketed || it.Ethereal {
		return Gray
	}

	return White
}

func (d *desc) isRune() bool {
	c := d.it.Code

	return len(c) == 3 && c[0] == 'r' && c[1] >= '0' && c[1] <= '3' && c[2] >= '0' && c[2] <= '9' && c != "rvs" && c != "rvl"
}

// baseName is the item's type name ("Monarch"), translated.
func (d *desc) baseName() string {
	key := d.base.NameStr
	if key == "" {
		key = d.base.Code
	}

	if s := d.t.Tr(key); s != "" {
		return s
	}

	return d.base.Name
}

// nameLines are the coloured name lines of the item: unique and set items
// have the item name and then the base item, rares two name words and the
// base item, magic items one line with prefix, base and suffix.
func (d *desc) nameLines() []Line {
	it, t := d.it, d.t
	base := d.baseName()
	col := d.qualityColor()

	var lines []Line

	add := func(s string, c Color) {
		if it.Personalized && len(lines) == 0 && it.PersonalName != "" {
			s = it.PersonalName + "'s " + s // verified format is the localized "%s's"; English only here
		}

		lines = append(lines, Line{s, c})
	}

	if !d.identified() && it.Quality >= d2s.QualityMagic {
		add(base, col)

		return lines
	}

	switch it.Quality {
	case d2s.QualityLow:
		base = t.format2("LowqualityFormat", "%0 %1", d.lowQualityName(), base)
	case d2s.QualityHigh:
		base = t.format2("HiqualityFormat", "%0 %1", t.tr("Hiquality"), base)
	}

	if rw := d.runeword(); it.Runeword && rw != nil {
		add(t.tr(rw.Key), Gold)
		lines = append(lines, Line{base, Gray})

		return lines
	}

	switch it.Quality {
	case d2s.QualityMagic:
		pre, suf := d.affixName(t.Prefixes, int(it.MagicPrefix)), d.affixName(t.Suffixes, int(it.MagicSuffix))
		add(t.format2("MagicFormat", "%0 %1 %2", pre, base, suf), Blue)
	case d2s.QualityRare, d2s.QualityCrafted:
		add(t.format2("RareFormat", "%0 %1", d.rareName(1), d.rareName(2)), col)
		lines = append(lines, Line{base, col})
	case d2s.QualityUnique:
		if int(it.UniqueID) < len(t.Uniques) {
			add(t.tr(t.Uniques[it.UniqueID].Index), Gold)
			lines = append(lines, Line{base, Gold})
		} else {
			add(base, Gold)
		}
	case d2s.QualitySet:
		if int(it.SetID) < len(t.SetItems) {
			add(t.tr(t.SetItems[it.SetID].Index), Green)
			// the base item of a set item is gold (recollection of the
			// original tooltip, unverified)
			lines = append(lines, Line{base, Gold})
		} else {
			add(base, Green)
		}
	default:
		add(base, col)
	}

	return lines
}

func (d *desc) lowQualityName() string {
	// ids 0..3 are Crude, Cracked, Damaged, Low Quality in the save
	names := [...]string{"Crude", "Cracked", "Damaged", "Low Quality"}
	if int(d.it.LowQualityID) < len(names) {
		return d.t.tr(names[d.it.LowQualityID])
	}

	return d.t.tr("Low Quality")
}

func (d *desc) affixName(list []AffixDef, id int) string {
	if id <= 0 || id >= len(list) {
		return ""
	}

	return d.t.tr(list[id].Name)
}

// rareName picks the first (n=1) or second word of a rare item name.
func (d *desc) rareName(n int) string {
	if n == 1 {
		return d.rareWord(d.t.RarePrefixes, int(d.it.RareName1), rarePrefixBase)
	}

	return d.rareWord(d.t.RareSuffix, int(d.it.RareName2), rareSuffixBase)
}

// the save numbers the rare name words from these bases: RareName1/2 are
// ids into rareprefix.txt / raresuffix.txt (checked against the sample
// save's rare rings by the tests).
const (
	rarePrefixBase = 156
	rareSuffixBase = 1
)

func (d *desc) rareWord(list []string, id, base int) string {
	i := id - base
	if i < 0 || i >= len(list) {
		return ""
	}

	return d.t.tr(list[i])
}

func (d *desc) runeword() *RunewordDef {
	i := int(d.it.RunewordID) - runewordIDBase
	if i < 0 || i >= len(d.t.Runewords) {
		return nil
	}

	return &d.t.Runewords[i]
}

// enhancement percentages

func (d *desc) defenseLine() (Line, bool) {
	b := d.base
	if b.MaxAC == 0 {
		return Line{}, false
	}

	flat := d.value(31)
	ed := d.value(16) // item_armor_percent
	def := d.it.Defense
	def = def*(100+ed)/100 + flat

	c := White
	if ed != 0 || flat != 0 || d.it.Ethereal || d.it.Quality == d2s.QualityHigh {
		c = Blue
	}

	return Line{d.t.tr("ItemStats1h") + " " + strconv.Itoa(def), c}, true
}

func (d *desc) damageRange(lo, hi int) (int, int, Color) {
	elo, ehi := d.value(statEnhDmgMin), d.value(statEnhDmgMax)
	flo, fhi := d.value(21), d.value(22)

	if d.it.Ethereal {
		lo, hi = lo*3/2, hi*3/2
	}

	c := White
	if d.it.Ethereal || elo != 0 || ehi != 0 || flo != 0 || fhi != 0 {
		c = Blue
	}

	lo = lo*(100+elo)/100 + flo
	hi = hi*(100+ehi)/100 + fhi

	if hi < lo {
		hi = lo
	}

	return lo, hi, c
}

func (d *desc) dmgLine(labelKey string, lo, hi int) Line {
	lo, hi, c := d.damageRange(lo, hi)

	return Line{fmt.Sprintf("%s %d %s %d", d.t.tr(labelKey), lo, d.t.tr("to"), hi), c}
}

func (d *desc) requirement(label string, need, have int, hasHero bool) Line {
	c := White
	if hasHero && have < need {
		c = Red
	}

	return Line{fmt.Sprintf("%s %d", d.t.tr(label), need), c}
}

func (d *desc) reqPercent() int { return d.value(91) }

func (d *desc) levelReq() int {
	t, it := d.t, d.it
	lv := d.base.ReqLevel

	upd := func(v int) {
		if v > lv {
			lv = v
		}
	}

	switch it.Quality {
	case d2s.QualityUnique:
		if int(it.UniqueID) < len(t.Uniques) {
			upd(t.Uniques[it.UniqueID].LevelReq)
		}
	case d2s.QualitySet:
		if int(it.SetID) < len(t.SetItems) {
			upd(t.SetItems[it.SetID].LevelReq)
		}
	case d2s.QualityMagic:
		upd(affixLevel(t.Prefixes, int(it.MagicPrefix)))
		upd(affixLevel(t.Suffixes, int(it.MagicSuffix)))
	case d2s.QualityRare, d2s.QualityCrafted:
		for i, a := range it.RareAffixes {
			if it.RareMask&(1<<uint(i)) == 0 {
				continue
			}

			if i%2 == 0 {
				upd(affixLevel(t.Prefixes, int(a)))
			} else {
				upd(affixLevel(t.Suffixes, int(a)))
			}
		}
	}

	for i := range it.Children {
		if cb := t.Bases[it.Children[i].Code]; cb != nil {
			upd(cb.ReqLevel)
		}
	}

	return lv
}

func affixLevel(list []AffixDef, id int) int {
	if id <= 0 || id >= len(list) {
		return 0
	}

	return list[id].LevelReq
}

// baseLines: defense, block, damage, quantity, durability, requirements.
func (d *desc) baseLines() []Line {
	t, it, b := d.t, d.it, d.base

	var lines []Line

	// potions, keys and the like have no stats but a description in the
	// real game, which comes from the item's name string; not modelled
	if l, ok := d.defenseLine(); ok {
		lines = append(lines, l)
	}

	if b.Armor && b.Block > 0 {
		blk := b.Block + d.value(20)
		lines = append(lines, Line{t.tr("ItemStats1r") + strconv.Itoa(blk) + "%", boolColor(d.value(20) != 0)})
	}

	if b.Weapon || (b.Armor && b.MaxDam > 0 && d.t.IsA(b.Type, "pala")) {
		lines = append(lines, d.damageLines()...)
	}

	if b.Stackable || it.Quantity > 0 {
		lines = append(lines, Line{t.tr("ItemStats1i") + " " + strconv.Itoa(int(it.Quantity)), White})
	}

	if dl, ok := d.durabilityLine(); ok {
		lines = append(lines, dl)
	}

	hero := d.ctx != nil && d.ctx.HasHero

	str, dex := b.ReqStr, b.ReqDex
	if str > 0 && it.Ethereal {
		str -= 10
	}

	if dex > 0 && it.Ethereal {
		dex -= 10
	}

	if p := d.reqPercent(); p != 0 {
		str, dex = str*(100+p)/100, dex*(100+p)/100
	}

	var cs, cd, cl int
	if hero {
		cs, cd, cl = d.ctx.Strength, d.ctx.Dexterity, d.ctx.CharLevel
	}

	if str > 1 {
		lines = append(lines, d.requirement("ItemStats1e", str, cs, hero))
	}

	if dex > 1 {
		lines = append(lines, d.requirement("ItemStats1f", dex, cd, hero))
	}

	if lv := d.levelReq(); lv > 1 {
		lines = append(lines, d.requirement("ItemStats1p", lv, cl, hero))
	}

	if cl := t.classOfType(b.Type, 0); cl != "" {
		if idx, ok := classCodes[cl]; ok {
			if c := t.class(idx); c != nil {
				col := White
				if hero && d.ctx.Class >= 0 && d.ctx.Class != idx {
					col = Red
				}

				lines = append(lines, Line{t.tr(c.ClassOnly), col})
			}
		}
	}

	return lines
}

func boolColor(modified bool) Color {
	if modified {
		return Blue
	}

	return White
}

func (d *desc) damageLines() []Line {
	b := d.base

	var lines []Line

	switch {
	case b.Armor: // paladin shields: smite damage
		lines = append(lines, d.dmgLine("ItemStats1o", b.MinDam, b.MaxDam))
	case b.TwoOnly:
		lines = append(lines, d.dmgLine("ItemStats1m", b.TwoMin, b.TwoMax))
	default:
		if b.MaxDam > 0 {
			lines = append(lines, d.dmgLine("ItemStats1l", b.MinDam, b.MaxDam))
		}

		if b.OneOrTwo && b.TwoMax > 0 {
			lines = append(lines, d.dmgLine("ItemStats1m", b.TwoMin, b.TwoMax))
		}
	}

	if b.MissMax > 0 {
		lines = append(lines, d.dmgLine("ItemStats1n", b.MissMin, b.MissMax))
	}

	return lines
}

func (d *desc) durabilityLine() (Line, bool) {
	it, b := d.it, d.base
	if b.NoDurability || it.MaxDurability == 0 || d.has(152) { // 152 item_indesctructible
		return Line{}, false
	}

	c := White
	if it.Durability == 0 {
		c = Red
	}

	return Line{fmt.Sprintf("%s %d %s %d", d.t.tr("ItemStats1d"), it.Durability, d.t.tr("ItemStats1j"), it.MaxDurability), c}, true
}

// propertyLines are the magical properties, blue.
func (d *desc) propertyLines() []Line {
	if !d.identified() && d.it.Quality >= d2s.QualityMagic {
		return []Line{{d.t.tr("ItemStats1b"), Red}}
	}

	var lines []Line
	for _, s := range d.t.StatLines(d.stats, d.ctx) {
		lines = append(lines, Line{s, Blue})
	}

	return lines
}

// footLines: ethereal and socket count.
func (d *desc) footLines() []Line {
	var lines []Line

	if d.it.Ethereal {
		lines = append(lines, Line{d.t.EtherealText, Blue})
	}

	if d.it.Socketed && d.it.TotalSockets > 0 {
		lines = append(lines, Line{fmt.Sprintf("%s (%d)", d.t.tr("Socketable"), d.it.TotalSockets), Blue})
	}

	return lines
}

// setLines: the item's active set bonuses (green), then the set with its
// pieces (worn: green, missing: red) and the active set wide bonuses.
func (d *desc) setLines() []Line {
	it, t := d.it, d.t
	if it.Quality != d2s.QualitySet || int(it.SetID) >= len(t.SetItems) {
		return nil
	}

	si := t.SetItems[it.SetID]
	set := t.Sets[si.Set]

	worn := map[string]bool{}
	if d.ctx != nil && d.ctx.WornSetItems != nil {
		worn = d.ctx.WornSetItems
	}

	if it.Location == d2s.LocationEquipped && !worn[si.Index] {
		w := map[string]bool{si.Index: true}
		for k, v := range worn {
			w[k] = v
		}

		worn = w
	}

	count := 0

	if set != nil {
		for _, name := range set.Items {
			if worn[name] {
				count++
			}
		}
	}

	var lines []Line

	// the item's own bonus lists, one per set bit; list n needs n+2 pieces
	ln := 0

	for i := 0; i < 5; i++ {
		if it.SetListMask&(1<<uint(i)) == 0 {
			continue
		}

		if ln < len(it.SetProperties) && count >= i+2 {
			for _, s := range t.StatLines(toStats(it.SetProperties[ln]), d.ctx) {
				lines = append(lines, Line{s, Green})
			}
		}

		ln++
	}

	if set == nil {
		return lines
	}

	lines = append(lines, Line{"", White}, Line{t.tr(set.Name), Gold})

	for _, name := range set.Items {
		c := Red
		if worn[name] {
			c = Green
		}

		lines = append(lines, Line{t.tr(name), c})
	}

	for i, specs := range set.Partial {
		if count >= i+2 {
			lines = append(lines, d.specLines(specs, Green)...)
		}
	}

	if len(set.Items) > 0 && count >= len(set.Items) {
		lines = append(lines, d.specLines(set.Full, Gold)...)
	}

	return lines
}

func (d *desc) specLines(specs []PropSpec, c Color) []Line {
	var stats []Stat
	for _, s := range specs {
		stats = append(stats, d.t.EvalProp(s)...)
	}

	var out []Line
	for _, s := range d.t.StatLines(Merge(stats), d.ctx) {
		out = append(out, Line{s, c})
	}

	return out
}

// classOfType is the class restriction code ("pal") of an item type or of
// the type it derives from.
func (t *Tables) classOfType(code string, depth int) string {
	if c := t.typeClass[code]; c != "" {
		return c
	}

	if depth > 16 {
		return ""
	}

	for _, p := range t.types[code] {
		if c := t.classOfType(p, depth+1); c != "" {
			return c
		}
	}

	return ""
}

// socketableLines describe a gem, rune or jewel-like item that can be put
// into sockets: the sentence, then its effect per kind of item.
func (d *desc) socketableLines() []Line {
	g := d.t.Gems[d.it.Code]
	if g == nil {
		return nil
	}

	lines := []Line{{d.t.tr("ExInsertSockets"), White}}

	for _, k := range []struct {
		label string
		specs []PropSpec
	}{{"GemXp3", g.Weapon}, {"GemXp1", g.Helm}, {"GemXp2", g.Shield}} {
		var stats []Stat
		for _, sp := range k.specs {
			stats = append(stats, d.t.EvalProp(sp)...)
		}

		for i, l := range d.t.StatLines(Merge(stats), d.ctx) {
			if i == 0 {
				l = d.t.tr(k.label) + " " + l
			}

			lines = append(lines, Line{l, Blue})
		}
	}

	return lines
}

// potionLines: the rejuvenation potions state what they heal.
func (d *desc) potionLines() []Line {
	switch d.it.Code {
	case "rvs":
		return []Line{{d.t.tr("ItemStatsrejuv1"), White}}
	case "rvl":
		return []Line{{d.t.tr("ItemStatsrejuv2"), White}}
	}

	return nil
}

// ItemStats returns the merged stats of an item: its own, its runeword's and
// the effects of what is socketed into it.
func (t *Tables) ItemStats(it *d2s.Item) []Stat {
	base := t.Bases[it.Code]
	if base == nil {
		return toStats(it.Properties)
	}

	d := &desc{t: t, it: it, base: base}
	d.collect()

	return d.stats
}
