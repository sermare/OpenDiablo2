package d2util

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// AsterToEmpty converts strings beginning with "*" to "", for use when handling columns where an asterix can be used to comment out entries
func AsterToEmpty(text string) string {
	if strings.HasPrefix(text, "*") {
		return ""
	}

	return text
}

// EmptyToZero converts empty strings to "0" and leaves non-empty strings as is,
// for use before converting numerical data which equates empty to zero
func EmptyToZero(text string) string {
	if text == "" || text == " " {
		return "0"
	}

	return text
}

// StringToInt converts a string to an integer
func StringToInt(text string) int {
	result, err := strconv.Atoi(text)
	if err != nil {
		panic(err)
	}

	return result
}

// SafeStringToInt converts a string to an integer, or returns -1 on falure

// StringToUint converts a string to a uint32
func StringToUint(text string) uint {
	result, err := strconv.ParseUint(text, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint(result)
}

// StringToUint8 converts a string to an uint8
func StringToUint8(text string) uint8 {
	result, err := strconv.Atoi(text)
	if err != nil {
		panic(err)
	}

	if result < 0 || result > 255 {
		panic(fmt.Sprintf("value %d out of range of byte", result))
	}

	return uint8(result)
}

// StringToInt8 converts a string to an int8
func StringToInt8(text string) int8 {
	result, err := strconv.Atoi(text)
	if err != nil {
		panic(err)
	}

	if result < -128 || result > 122 {
		panic(fmt.Sprintf("value %d out of range of a signed byte", result))
	}

	return int8(result)
}

// Utf16BytesToString converts a utf16 byte array to string
func Utf16BytesToString(b []byte) (string, error) {
	if len(b)%2 != 0 {
		return "", fmt.Errorf("must have even length byte slice")
	}

	u16s := make([]uint16, 1)

	ret := &bytes.Buffer{}

	b8buf := make([]byte, 4)

	lb := len(b)
	for i := 0; i < lb; i += 2 {
		// nolint:gomnd // byte operation
		u16s[0] = uint16(b[i]) + (uint16(b[i+1]) << 8)
		r := utf16.Decode(u16s)
		n := utf8.EncodeRune(b8buf, r[0])
		ret.Write(b8buf[:n])
	}

	return ret.String(), nil
}

// SplitIntoLinesWithMaxWidth splits the given string into lines considering the given maxChars.
// Widths are counted in runes, with glyph codes above 0xFF (double byte CJK glyphs, see
// d2locale) counting as two columns. Text that contains such glyphs, or a first word wider than
// maxChars, is cut by glyph because those scripts do not separate words with blanks.
func SplitIntoLinesWithMaxWidth(fullSentence string, maxChars int) []string {
	lines := make([]string, 0)
	line := ""
	totalLength := 0
	words := strings.Split(fullSentence, " ")

	if textWidth(words[0]) > maxChars || hasWideGlyph(fullSentence) {
		return splitCjkIntoChunks(fullSentence, maxChars)
	}

	for _, word := range words {
		totalLength += 1 + textWidth(word)
		if totalLength > maxChars {
			totalLength = textWidth(word)

			lines = append(lines, line)
			line = ""
		} else {
			line += " "
		}

		line += word
	}

	if len(line) > 0 {
		lines = append(lines, line)
	}

	return lines
}

const wideGlyphStart = 0x100

func hasWideGlyph(s string) bool {
	for _, r := range s {
		if r >= wideGlyphStart {
			return true
		}
	}

	return false
}

func runeWidth(r rune) int {
	if r >= wideGlyphStart {
		return 2 // nolint:gomnd // double byte glyph
	}

	return 1
}

func textWidth(s string) int {
	w := 0
	for _, r := range s {
		w += runeWidth(r)
	}

	return w
}

// splitCjkIntoChunks cuts str into lines of at most chars columns, never splitting a glyph
func splitCjkIntoChunks(str string, chars int) []string {
	chunks := make([]string, 0)

	var cur []rune

	width := 0

	for _, ch := range str {
		w := runeWidth(ch)
		if width+w > chars && len(cur) > 0 {
			chunks = append(chunks, string(cur))
			cur, width = cur[:0], 0
		}

		cur = append(cur, ch)
		width += w
	}

	if len(cur) > 0 {
		chunks = append(chunks, string(cur))
	}

	return chunks
}
