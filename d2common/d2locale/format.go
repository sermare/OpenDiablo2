package d2locale

import (
	"strconv"
	"strings"
)

// FormatInt groups the digits of n with the locale's thousands separator.
func (l Locale) FormatInt(n int) string {
	s := strconv.FormatInt(int64(n), 10)
	neg := strings.HasPrefix(s, "-")

	if neg {
		s = s[1:]
	}

	var b strings.Builder

	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteString(l.ThousandSep)
		}

		b.WriteRune(c)
	}

	if neg {
		return "-" + b.String()
	}

	return b.String()
}

// FormatFloat formats f with the given number of decimals using the locale separators.
func (l Locale) FormatFloat(f float64, decimals int) string {
	s := strconv.FormatFloat(f, 'f', decimals, 64)
	intPart, frac := s, ""

	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart, frac = s[:i], s[i+1:]
	}

	neg := strings.HasPrefix(intPart, "-")
	n, _ := strconv.Atoi(strings.TrimPrefix(intPart, "-"))
	out := l.FormatInt(n)

	if neg {
		out = "-" + out
	}

	if frac != "" {
		out += l.DecimalSep + frac
	}

	return out
}
