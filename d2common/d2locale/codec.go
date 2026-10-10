package d2locale

// Strings in the retail .tbl files are in the code page of their language, and the locale fonts
// are indexed by the same raw codes. To render without code page tables the engine keeps text as
// "glyph code strings": every single byte b becomes the rune U+00bb, and in double byte locales a
// lead byte plus trail byte becomes one rune (lead<<8 | trail). ASCII is unchanged.
// That the CJK fonts index glyphs by lead<<8|trail is UNVERIFIED (no CJK install available).

// IsLeadByte reports whether b starts a two byte character in the locale.
func (l Locale) IsLeadByte(b byte) bool {
	if l.Kind != DoubleByte {
		return false
	}

	if l.Codepage == 932 {
		return (b >= 0x81 && b <= 0x9F) || (b >= 0xE0 && b <= 0xFC)
	}

	return b >= 0x81 && b <= 0xFE // 936, 949, 950
}

// Decode converts raw table bytes into a glyph code string.
func (l Locale) Decode(raw []byte) string {
	out := make([]rune, 0, len(raw))

	for i := 0; i < len(raw); i++ {
		b := raw[i]
		if l.IsLeadByte(b) && i+1 < len(raw) && raw[i+1] != 0 {
			out = append(out, rune(b)<<8|rune(raw[i+1]))
			i++

			continue
		}

		out = append(out, rune(b))
	}

	return string(out)
}

// Encode is the inverse of Decode. Runes above 0xFFFF are replaced with '?'.
func (l Locale) Encode(s string) []byte {
	out := make([]byte, 0, len(s))

	for _, r := range s {
		switch {
		case r > 0xFFFF:
			out = append(out, '?')
		case r > 0xFF:
			out = append(out, byte(r>>8), byte(r))
		default:
			out = append(out, byte(r))
		}
	}

	return out
}

// ToUnicode converts a glyph code string to real Unicode where the code page is handled (1252
// and 1251; other code pages are returned unchanged). Intended for logs, not for rendering.
func (l Locale) ToUnicode(s string) string {
	switch l.Codepage {
	case 1252:
		return mapRunes(s, func(r rune) rune {
			if v, ok := cp1252Table[r]; ok {
				return v
			}

			return r
		})
	case 1251:
		return mapRunes(s, func(r rune) rune {
			switch {
			case r >= 0xC0 && r <= 0xFF:
				return r - 0xC0 + 0x0410
			case r == 0xA8:
				return 0x0401
			case r == 0xB8:
				return 0x0451
			}

			return r
		})
	}

	return s
}

func mapRunes(s string, f func(rune) rune) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		out = append(out, f(r))
	}

	return string(out)
}

var cp1252Table = map[rune]rune{
	0x80: 0x20AC, 0x82: 0x201A, 0x83: 0x0192, 0x84: 0x201E, 0x85: 0x2026, 0x86: 0x2020, 0x87: 0x2021,
	0x88: 0x02C6, 0x89: 0x2030, 0x8A: 0x0160, 0x8B: 0x2039, 0x8C: 0x0152, 0x8E: 0x017D,
	0x91: 0x2018, 0x92: 0x2019, 0x93: 0x201C, 0x94: 0x201D, 0x95: 0x2022, 0x96: 0x2013, 0x97: 0x2014,
	0x98: 0x02DC, 0x99: 0x2122, 0x9A: 0x0161, 0x9B: 0x203A, 0x9C: 0x0153, 0x9E: 0x017E, 0x9F: 0x0178,
}
