package d2s

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ErrCorrupt is returned when a damaged save made the parser fail in a way it
// did not anticipate (an internal panic is turned into this error).
var ErrCorrupt = errors.New("d2s: corrupt save")

// VersionName names the game release that writes a given file version.
// Versions below MinVersion use the legacy layouts (see docs/save-compat.md);
// the code reads and writes only MinVersion..MaxVersion. The names of versions
// other than 0x60 come from community format notes and are unverified.
func VersionName(v uint32) string {
	switch {
	case v == 0x47:
		return "1.00-1.05, legacy layout"
	case v == 0x57:
		return "1.06-1.07, legacy layout"
	case v == 0x59:
		return "1.08-1.09, legacy layout"
	case v == 0x5C:
		return "1.10-1.14a"
	case v == 0x5D || v == 0x5E || v == 0x5F:
		return "1.10-1.14a, unverified"
	case v == 0x60:
		return "1.14b/d"
	case v > MaxVersion:
		return "newer than 1.14, e.g. Resurrected"
	}

	return "unknown version"
}

// IsLegacyVersion reports whether v is an older layout the parser rejects and
// docs/save-compat.md describes a migration for.
func IsLegacyVersion(v uint32) bool { return v < MinVersion }

// Repair makes the size field and the checksum of a save consistent with its
// bytes, so a save whose checksum went stale (hand edit, partial copy) can be
// parsed. It never changes anything else; fixes lists what was changed. The
// magic number must be right and the header complete, otherwise the file is
// not a save and an error is returned. data is not modified.
func Repair(data []byte) (fixed []byte, fixes []string, err error) {
	if len(data) < HeaderSize {
		return nil, nil, ErrTooShort
	}

	if binary.LittleEndian.Uint32(data) != Magic {
		return nil, nil, ErrBadMagic
	}

	fixed = append([]byte(nil), data...)
	le := binary.LittleEndian

	if int(le.Uint32(fixed[8:])) != len(fixed) {
		fixes = append(fixes, fmt.Sprintf("size field %d -> %d", le.Uint32(fixed[8:]), len(fixed)))
		le.PutUint32(fixed[8:], uint32(len(fixed)))
	}

	if sum := Checksum(fixed); le.Uint32(fixed[checksumOffset:]) != sum {
		fixes = append(fixes, fmt.Sprintf("checksum %08X -> %08X", le.Uint32(fixed[checksumOffset:]), sum))
		le.PutUint32(fixed[checksumOffset:], sum)
	}

	return fixed, fixes, nil
}

// ParseRepair is Parse after Repair: a bad size or checksum is fixed (and
// reported in fixes) instead of rejected. Damage past the header still fails.
func ParseRepair(data []byte, tables *ItemTables) (c *Character, fixes []string, err error) {
	fixed, fixes, err := Repair(data)
	if err != nil {
		return nil, nil, err
	}

	c, err = Parse(fixed, tables)

	return c, fixes, err
}

// Unsupported lists the parts of a parsed save that Write carries over as raw
// bytes without understanding them, so a caller can tell what a round trip
// preserves only blindly. An empty result means every byte is modelled.
func Unsupported(c *Character) []string {
	var out []string

	if c == nil || c.Header == nil {
		return out
	}

	if len(c.Trailing) > 0 {
		out = append(out, fmt.Sprintf("trailing data: %d bytes after the last known section", len(c.Trailing)))
	}

	if c.HasCorpse {
		out = append(out, "corpse header: 12 bytes kept verbatim (meaning unverified)")
	}

	if c.Header.Version != 0x60 {
		out = append(out, fmt.Sprintf("version 0x%X is only verified against 0x60", c.Header.Version))
	}

	return out
}

// safely runs f and turns a panic into ErrCorrupt.
func safely(f func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrCorrupt, r)
		}
	}()

	return f()
}
