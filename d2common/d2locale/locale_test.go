package d2locale

import (
	"reflect"
	"strings"
	"testing"
)

func TestAllLocalesUnique(t *testing.T) {
	codes, dirs, bytes := map[string]bool{}, map[string]bool{}, map[byte]bool{}

	if len(All()) != 12 {
		t.Fatalf("want 12 locales, got %d", len(All()))
	}

	for _, l := range All() {
		if codes[l.Code] || dirs[l.Dir] || bytes[l.UseByte] {
			t.Errorf("duplicate entry %+v", l)
		}

		codes[l.Code], dirs[l.Dir], bytes[l.UseByte] = true, true, true

		if len(l.FontDirs) == 0 || l.Codepage == 0 {
			t.Errorf("%s incomplete", l.Code)
		}
	}
}

func TestParse(t *testing.T) {
	for in, want := range map[string]string{
		"enUS": "enUS", "de-DE": "deDE", "ru_ru": "ruRU", "DEU": "deDE", "ja": "jaJP", "zhtw": "zhTW", "SIN": "zhCN", " pt ": "ptBR",
	} {
		got, ok := Parse(in)
		if !ok || got.Code != want {
			t.Errorf("Parse(%q)=%q,%v want %s", in, got.Code, ok, want)
		}
	}

	for _, bad := range []string{"", "xx", "klingon"} {
		if _, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) should fail", bad)
		}
	}
}

func TestFromUseByte(t *testing.T) {
	for _, l := range All() {
		got, ok := FromUseByte(l.UseByte)
		if !ok || got.Code != l.Code {
			t.Errorf("byte %d -> %s", l.UseByte, got.Code)
		}
	}

	if l, ok := FromUseByte(0x0C); !ok || l.Code != "enUS" {
		t.Error("0x0C should be English")
	}

	if _, ok := FromUseByte(0x55); ok {
		t.Error("unknown byte")
	}
}

func fakeInstall(paths ...string) func(string) bool {
	set := map[string]bool{}
	for _, p := range paths {
		set[p] = true
	}

	return func(p string) bool { return set[p] }
}

func TestChooseAndDetect(t *testing.T) {
	exists := fakeInstall("/data/local/lng/ENG/string.tbl", "/data/local/lng/DEU/string.tbl", "/data/local/FONT/LATIN/font16.tbl")

	if got := Detect(exists); len(got) != 2 || got[0].Code != "enUS" || got[1].Code != "deDE" {
		t.Errorf("detect: %+v", got)
	}

	cases := []struct {
		req  string
		use  int
		want string
		note bool
	}{
		{"", 0, "enUS", false},
		{"auto", 2, "deDE", false},
		{"deDE", 0, "deDE", false},
		{"frFR", 2, "deDE", true}, // not installed: install language
		{"nonsense", 0, "enUS", true},
		{"", -1, "enUS", false}, // use file unreadable
		{"", 0x77, "enUS", false},
	}

	for _, c := range cases {
		got, note := Choose(c.req, c.use, exists)
		if got.Code != c.want || (note != "") != c.note {
			t.Errorf("Choose(%q,%d)=%s,%q", c.req, c.use, got.Code, note)
		}
	}
}

func TestResolveFontDir(t *testing.T) {
	zh, _ := Parse("zhCN")

	if got := zh.ResolveFontDir(fakeInstall("/data/local/FONT/CHI/font16.tbl")); got != "CHI" {
		t.Errorf("fallback font dir: %s", got)
	}

	if got := zh.ResolveFontDir(fakeInstall()); got != "SIN" {
		t.Errorf("default font dir: %s", got)
	}

	pl, _ := Parse("plPL")
	if pl.FontDirs[0] != "LATIN2" || pl.LabelModifier != 1 {
		t.Error("plPL")
	}

	ru, _ := Parse("ruRU")
	if ru.FontDirs[0] != "CYR" {
		t.Error("ruRU")
	}
}

func TestCodecRoundTrip(t *testing.T) {
	samples := map[string][]byte{
		"deDE": []byte("Gr\xfc\xdfe \xe4\xf6 \xffc3x"),
		"plPL": []byte("\xa3\xf3d\xbf \xea\xb3"),
		"ruRU": []byte("\xcf\xf0\xe8\xe2\xe5\xf2 \xa8"),
		"jaJP": {0x82, 0xa0, 'A', 0xb1, 0x83, 0x41}, // hiragana, A, half-width katakana, katakana
		"koKR": {0xb0, 0xa1, ' ', 'x', 0xc7, 0xd1},
		"zhCN": {0xc4, 0xe3, 0xba, 0xc3, '!'},
		"zhTW": {0xa7, 0x41, 0xa6, 0x6e},
	}

	for code, raw := range samples {
		l, _ := Parse(code)
		s := l.Decode(raw)

		if got := l.Encode(s); !reflect.DeepEqual(got, raw) {
			t.Errorf("%s: % x -> % x", code, raw, got)
		}

		if !strings.ContainsRune(s, 'x') && code != "jaJP" && code != "ruRU" && code != "plPL" && code != "zhCN" && code != "zhTW" {
			t.Errorf("%s: ascii lost: %q", code, s)
		}
	}

	ja, _ := Parse("jaJP")
	if got := []rune(ja.Decode([]byte{0x82, 0xa0, 'A', 0xb1})); !reflect.DeepEqual(got, []rune{0x82a0, 'A', 0xb1}) {
		t.Errorf("sjis decode: %x", got)
	}

	de, _ := Parse("deDE")
	if got := de.ToUnicode(de.Decode([]byte("Gr\xfc\xdfe \x80"))); got != "Grüße €" {
		t.Errorf("cp1252: %q", got)
	}

	ru, _ := Parse("ruRU")
	if got := ru.ToUnicode(ru.Decode([]byte("\xcf\xf0\xe8"))); got != "При" {
		t.Errorf("cp1251: %q", got)
	}
}

func TestFormat(t *testing.T) {
	en, _ := Parse("enUS")
	de, _ := Parse("deDE")
	fr, _ := Parse("frFR")

	cases := []struct {
		l    Locale
		n    int
		want string
	}{
		{en, 0, "0"}, {en, 999, "999"}, {en, 1000, "1,000"}, {en, 2500000, "2,500,000"}, {en, -12345, "-12,345"},
		{de, 1234567, "1.234.567"}, {fr, 12345, "12 345"},
	}

	for _, c := range cases {
		if got := c.l.FormatInt(c.n); got != c.want {
			t.Errorf("%s FormatInt(%d)=%q want %q", c.l.Code, c.n, got, c.want)
		}
	}

	if got := de.FormatFloat(1234.5, 2); got != "1.234,50" {
		t.Errorf("de float %q", got)
	}

	if got := en.FormatFloat(-0.5, 1); got != "-0.5" {
		t.Errorf("en float %q", got)
	}
}

func TestItemNames(t *testing.T) {
	en, _ := Parse("enUS")
	ja, _ := Parse("jaJP")

	if got := en.MagicName("Sturdy", "Cap", "of the Fox"); got != "Sturdy Cap of the Fox" {
		t.Error(got)
	}

	if got := en.MagicName("", "Cap", "of the Fox"); got != "Cap of the Fox" {
		t.Error(got)
	}

	if got := en.MagicName("Sturdy", "Cap", ""); got != "Sturdy Cap" {
		t.Error(got)
	}

	if got := en.RareName("beast bite", "grip", "Gloves"); got != "Beast Bite Grip\nGloves" {
		t.Error(got)
	}

	if got := ja.MagicName("A", "B", "C"); got != "ABC" {
		t.Error(got)
	}

	alt := en
	alt.Order = BaseThenAffixes

	if got := alt.MagicName("Sturdy", "Cap", "of the Fox"); got != "Cap Sturdy of the Fox" {
		t.Error(got)
	}
}

func width(s string) int {
	w := 0

	for _, r := range s {
		if r > 0xFF {
			w += 2
		} else {
			w++
		}
	}

	return w
}

func TestWrap(t *testing.T) {
	en, _ := Parse("enUS")

	got := en.WrapText("the quick brown fox\njumps", 10, width)
	want := []string{"the quick", "brown fox", "jumps"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("%q", got)
	}

	if got := en.WrapText("abcdefghijkl mn", 5, width); !reflect.DeepEqual(got, []string{"abcde", "fghij", "kl mn"}) {
		t.Errorf("long word: %q", got)
	}

	if got := en.WrapText("short", 0, width); len(got) != 1 {
		t.Error("no wrap")
	}

	ja, _ := Parse("jaJP")
	cjk := ja.Decode([]byte{0x82, 0xa0, 0x82, 0xa2, 0x82, 0xa4, 0x82, 0xa6, 0x82, 0xa8})

	for _, line := range ja.WrapText(cjk, 4, width) {
		if width(line) > 4 {
			t.Errorf("line too wide: %q", line)
		}
	}

	if got := ja.WrapText(cjk, 4, width); len(got) != 3 {
		t.Errorf("cjk lines: %q", got)
	}
}
