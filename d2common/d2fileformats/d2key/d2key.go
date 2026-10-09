package d2key

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// File format constants.
const (
	Magic       = 0x5357
	Version     = 0x25
	FileSize    = 0x47a
	HeaderSize  = 6
	RecordSize  = 20
	RecordCount = 57

	// Unbound is the key code stored for "no key".
	Unbound = 0xffff
)

// Record is one command binding.
type Record struct {
	Command   uint32
	Primary   uint16
	FlagA     uint16
	Secondary uint16
	FlagB     uint32
}

// File is a parsed default.key.
type File struct {
	Records []Record
}

// Parse decodes a default.key image.
func Parse(data []byte) (*File, error) {
	if len(data) != FileSize {
		return nil, fmt.Errorf("d2key: size %d, want %d", len(data), FileSize)
	}

	le := binary.LittleEndian
	if le.Uint16(data[0:]) != Magic || le.Uint16(data[2:]) != Version || le.Uint16(data[4:]) != FileSize {
		return nil, errors.New("d2key: bad header")
	}

	f := &File{Records: make([]Record, RecordCount)}

	for i := range f.Records {
		r := data[HeaderSize+i*RecordSize : HeaderSize+(i+1)*RecordSize]
		rec := Record{
			Command:   le.Uint32(r[0:]),
			Primary:   le.Uint16(r[4:]),
			FlagA:     le.Uint16(r[6:]),
			Secondary: le.Uint16(r[14:]),
			FlagB:     le.Uint32(r[16:]),
		}

		if uint32(le.Uint16(r[10:])) != rec.Command {
			return nil, fmt.Errorf("d2key: record %d: repeated command id mismatch", i)
		}

		f.Records[i] = rec
	}

	return f, nil
}

// Marshal encodes the file back to its on-disk form.
func (f *File) Marshal() []byte {
	le := binary.LittleEndian
	out := make([]byte, FileSize)
	le.PutUint16(out[0:], Magic)
	le.PutUint16(out[2:], Version)
	le.PutUint16(out[4:], FileSize)

	for i, rec := range f.Records {
		if i >= RecordCount {
			break
		}

		r := out[HeaderSize+i*RecordSize:]
		le.PutUint32(r[0:], rec.Command)
		le.PutUint16(r[4:], rec.Primary)
		le.PutUint16(r[6:], rec.FlagA)
		le.PutUint16(r[10:], uint16(rec.Command))
		le.PutUint16(r[14:], rec.Secondary)
		le.PutUint32(r[16:], rec.FlagB)
	}

	return out
}

// Lookup returns the record for a command id.
func (f *File) Lookup(command uint32) (Record, bool) {
	for _, r := range f.Records {
		if r.Command == command {
			return r, true
		}
	}

	return Record{}, false
}

// IsKeyboard reports whether a code is a real Windows virtual-key code
// (as opposed to unbound or one of the game's 0x100+ mouse codes).
func IsKeyboard(code uint16) bool {
	return code != Unbound && code < 0x100
}
