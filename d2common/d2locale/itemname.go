package d2locale

import "strings"

// MagicName composes a magic item name from already translated parts. Either affix may be
// empty. The enUS order (verified from the English game) is "prefix base suffix".
func (l Locale) MagicName(prefix, base, suffix string) string {
	parts := make([]string, 0, 3)

	add := func(s string) {
		if s != "" {
			parts = append(parts, s)
		}
	}

	if l.Order == BaseThenAffixes {
		add(base)
		add(prefix)
		add(suffix)
	} else {
		add(prefix)
		add(base)
		add(suffix)
	}

	return strings.Join(parts, l.NameSep)
}

// RareName composes a rare item name "<prefix> <suffix>\n<base>", with the two generated words
// title cased in alphabets that have case.
func (l Locale) RareName(prefix, suffix, base string) string {
	if l.TitleCaseRare {
		prefix, suffix = titleLatin(prefix), titleLatin(suffix)
	}

	return prefix + l.NameSep + suffix + "\n" + base
}

// titleLatin upper cases the first ASCII letter of each blank separated word; accented letters
// of the code pages are left alone.
func titleLatin(s string) string {
	b := []rune(s)
	start := true

	for i, r := range b {
		if start && r >= 'a' && r <= 'z' {
			b[i] = r - 'a' + 'A'
		}

		start = r == ' '
	}

	return string(b)
}
