package d2locale

import "strings"

// WrapText breaks text into lines no wider than maxWidth as measured by measure. Existing
// newlines are kept. Blank separated scripts break at blanks; NoSpaceWrap locales (Chinese,
// Japanese) may also break between any two double byte glyphs. A single unbreakable run wider
// than maxWidth is split per glyph as a last resort. maxWidth <= 0 disables wrapping.
func (l Locale) WrapText(text string, maxWidth int, measure func(string) int) []string {
	var lines []string

	for _, para := range strings.Split(text, "\n") {
		lines = append(lines, l.wrapLine(para, maxWidth, measure)...)
	}

	return lines
}

func (l Locale) wrapLine(s string, maxWidth int, measure func(string) int) []string {
	if maxWidth <= 0 || measure(s) <= maxWidth {
		return []string{s}
	}

	var (
		lines []string
		cur   string
	)

	flush := func() {
		lines = append(lines, strings.TrimRight(cur, " "))
		cur = ""
	}

	for _, tok := range l.tokens(s) {
		if cur != "" && measure(strings.TrimRight(cur+tok, " ")) > maxWidth {
			flush()
		}

		if measure(strings.TrimRight(tok, " ")) > maxWidth { // oversized run: split per glyph
			for _, r := range tok {
				g := string(r)
				if cur != "" && measure(cur+g) > maxWidth {
					flush()
				}

				cur += g
			}

			continue
		}

		cur += tok
	}

	if cur != "" {
		flush()
	}

	return lines
}

// tokens splits s into breakable units; trailing blanks stay attached to their word.
func (l Locale) tokens(s string) []string {
	var (
		out []string
		cur strings.Builder
	)

	flush := func() {
		if cur.Len() > 0 {
			out = append(out, cur.String())
			cur.Reset()
		}
	}

	prevBlank := false

	for _, r := range s {
		switch {
		case r == ' ':
			cur.WriteRune(r)

			prevBlank = true
		case l.NoSpaceWrap && r > 0xFF:
			flush()
			out = append(out, string(r))

			prevBlank = false
		default:
			if prevBlank {
				flush()
			}

			cur.WriteRune(r)

			prevBlank = false
		}
	}

	flush()

	return out
}
