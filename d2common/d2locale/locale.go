package d2locale

import (
	"strings"
)

// CharsetKind says how the bytes of a locale's string tables map to font glyph codes.
type CharsetKind int

const (
	// SingleByte locales (Latin, Latin-2, Cyrillic): every byte of a .tbl string is one glyph code.
	SingleByte CharsetKind = iota
	// DoubleByte locales (CJK): lead bytes start a two byte glyph code.
	DoubleByte
)

// NameOrder is the order in which the parts of a magic item name are composed.
type NameOrder int

const (
	// PrefixBaseSuffix gives "Sturdy Cap of the Fox" (the English order).
	PrefixBaseSuffix NameOrder = iota
	// BaseThenAffixes gives "Cap Sturdy of the Fox" (noun first). No retail locale is verified
	// to need it; kept so the order can be changed per locale from data.
	BaseThenAffixes
)

// Locale describes one retail language.
type Locale struct {
	// Code is the Blizzard style tag, e.g. "deDE".
	Code string
	// Dir is the directory token used below data/local/lng and data/local/ui, e.g. "DEU".
	Dir string
	// UseByte is the value of the byte stored in data/local/use for this language.
	UseByte byte
	// FontDirs are candidate directories below data/local/font, first existing one wins.
	FontDirs []string
	// Codepage is the Windows code page of the .tbl text.
	Codepage int
	Kind     CharsetKind
	// LabelModifier is the offset applied to numeric "#N" string ids.
	LabelModifier int
	// ThousandSep and DecimalSep are used by FormatInt/FormatFloat. Unverified against retail
	// for all but enUS.
	ThousandSep string
	DecimalSep  string
	// Order is the magic item name composition order.
	Order NameOrder
	// NameSep joins prefix, base and suffix. CJK languages may not use blanks (unverified).
	NameSep string
	// TitleCaseRare says rare names are title cased (alphabets with case only).
	TitleCaseRare bool
	// NoSpaceWrap marks scripts that may break between any two glyphs.
	NoSpaceWrap bool
}

// All returns the supported locales in a stable order (enUS first).
func All() []Locale {
	return []Locale{
		{Code: "enUS", Dir: "ENG", UseByte: 0x00, FontDirs: []string{"LATIN"}, Codepage: 1252, ThousandSep: ",", DecimalSep: ".", NameSep: " ", TitleCaseRare: true},
		{Code: "deDE", Dir: "DEU", UseByte: 0x02, FontDirs: []string{"LATIN"}, Codepage: 1252, ThousandSep: ".", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		{Code: "esES", Dir: "ESP", UseByte: 0x01, FontDirs: []string{"LATIN"}, Codepage: 1252, ThousandSep: ".", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		{Code: "frFR", Dir: "FRA", UseByte: 0x03, FontDirs: []string{"LATIN"}, Codepage: 1252, ThousandSep: " ", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		{Code: "itIT", Dir: "ITA", UseByte: 0x05, FontDirs: []string{"LATIN"}, Codepage: 1252, ThousandSep: ".", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		{Code: "jaJP", Dir: "JPN", UseByte: 0x06, FontDirs: []string{"JPN"}, Codepage: 932, Kind: DoubleByte, ThousandSep: ",", DecimalSep: ".", NameSep: "", NoSpaceWrap: true},
		{Code: "koKR", Dir: "KOR", UseByte: 0x07, FontDirs: []string{"KOR"}, Codepage: 949, Kind: DoubleByte, ThousandSep: ",", DecimalSep: ".", NameSep: " "},
		{Code: "plPL", Dir: "POL", UseByte: 0x0A, FontDirs: []string{"LATIN2"}, Codepage: 1250, LabelModifier: 1, ThousandSep: " ", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		{Code: "ptBR", Dir: "POR", UseByte: 0x04, FontDirs: []string{"LATIN"}, Codepage: 1252, ThousandSep: ".", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		{Code: "ruRU", Dir: "RUS", UseByte: 0x0B, FontDirs: []string{"CYR"}, Codepage: 1251, ThousandSep: " ", DecimalSep: ",", NameSep: " ", TitleCaseRare: true},
		// "SIN" is believed to be the simplified Chinese build and "CHI" the traditional one
		// (unverified); their font directories fall back to each other.
		{Code: "zhCN", Dir: "SIN", UseByte: 0x08, FontDirs: []string{"SIN", "CHI", "LATIN"}, Codepage: 936, Kind: DoubleByte, ThousandSep: ",", DecimalSep: ".", NameSep: "", NoSpaceWrap: true},
		{Code: "zhTW", Dir: "CHI", UseByte: 0x09, FontDirs: []string{"CHI", "SIN"}, Codepage: 950, Kind: DoubleByte, ThousandSep: ",", DecimalSep: ".", NameSep: "", NoSpaceWrap: true},
	}
}

// Default is the locale used when nothing else is known.
func Default() Locale { return All()[0] }

// FromUseByte finds the locale for the byte of data/local/use. Unknown values are not found.
func FromUseByte(b byte) (Locale, bool) {
	for _, l := range All() {
		if l.UseByte == b {
			return l, true
		}
	}

	if b == 0x0C { // second English slot in the retail language table
		return Default(), true
	}

	return Locale{}, false
}

// Parse accepts "deDE", "de-DE", "de_de", "DEU" (directory token) or "de".
func Parse(s string) (Locale, bool) {
	n := strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(strings.TrimSpace(s)))
	if n == "" {
		return Locale{}, false
	}

	for _, l := range All() {
		if n == strings.ToLower(l.Code) || n == strings.ToLower(l.Dir) {
			return l, true
		}
	}

	if len(n) == 2 {
		for _, l := range All() {
			if strings.HasPrefix(strings.ToLower(l.Code), n) {
				return l, true
			}
		}
	}

	return Locale{}, false
}

// StringTablePath returns the path of the named .tbl for this locale.
func (l Locale) StringTablePath(name string) string {
	return "/data/local/lng/" + l.Dir + "/" + name
}

// Installed reports whether the locale's main string table is reachable with exists.
func (l Locale) Installed(exists func(path string) bool) bool {
	return exists(l.StringTablePath("string.tbl"))
}

// Detect returns the locales whose string.tbl exists according to exists.
func Detect(exists func(path string) bool) []Locale {
	var out []Locale

	for _, l := range All() {
		if l.Installed(exists) {
			out = append(out, l)
		}
	}

	return out
}

// ResolveFontDir returns the first font directory of the locale that has font16.tbl.
func (l Locale) ResolveFontDir(exists func(path string) bool) string {
	for _, d := range l.FontDirs {
		if exists("/data/local/FONT/" + d + "/font16.tbl") {
			return d
		}
	}

	return l.FontDirs[0]
}

// Choose picks the locale to run in: the requested one when installed, otherwise the install's
// own (use byte, negative if unknown) language, otherwise enUS. note explains a fallback.
func Choose(requested string, useByte int, exists func(path string) bool) (loc Locale, note string) {
	base := Default()

	if useByte >= 0 && useByte <= 0xFF {
		if l, ok := FromUseByte(byte(useByte)); ok {
			base = l
		}
	}

	if strings.TrimSpace(requested) == "" || strings.EqualFold(requested, "auto") {
		return base, ""
	}

	want, ok := Parse(requested)
	if !ok {
		return base, "unknown language " + requested + ", using " + base.Code
	}

	if !want.Installed(exists) {
		return base, "language " + want.Code + " is not installed, using " + base.Code
	}

	return want, ""
}
