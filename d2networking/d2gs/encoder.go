package d2gs

import "fmt"

// Validate checks that b is exactly one well-formed packet for dir, using
// the same rules as the Decoder. Client packets whose size is unverified are
// accepted if they are within MaxPacketSize and have a handled id.
func Validate(dir Direction, b []byte) error {
	if len(b) == 0 {
		return ErrShort
	}
	if len(b) > MaxPacketSize {
		return ErrTooLarge
	}
	if dir == ServerToClient {
		n, need, err := serverLength(b)
		if err != nil {
			return err
		}
		if need > 0 {
			return ErrShort
		}
		if n != len(b) {
			return fmt.Errorf("%w: id %#x has length %d, buffer is %d", ErrWrongSize, b[0], n, len(b))
		}
		return nil
	}
	n, kind := ExpectedSize(b[0], dir)
	switch kind {
	case SizeInvalid:
		return ErrInvalidID
	case SizeFixed:
		if n != len(b) {
			return fmt.Errorf("%w: id %#x wants %d, got %d", ErrWrongSize, b[0], n, len(b))
		}
	}
	return nil
}

// Chunk is one SCMD_QueueServerPacket chunk: up to ChunkSize payload bytes.
type Chunk struct{ Data []byte }

// Encoder queues validated packets into 512-byte chunks the way the server's
// per-client builder list does, and flushes them as one byte stream.
//
// Unverified: how the original treats a packet that does not fit the tail of
// the current chunk. We start a new chunk (packets are never split); the
// packets of 0x201..0x204 bytes the size cap allows get a chunk of their own.
type Encoder struct {
	dir    Direction
	chunks []Chunk
}

// NewEncoder returns an encoder for dir.
func NewEncoder(dir Direction) *Encoder { return &Encoder{dir: dir} }

// Add validates and queues one packet.
func (e *Encoder) Add(pkt []byte) error {
	if err := Validate(e.dir, pkt); err != nil {
		return err
	}
	if n := len(e.chunks); n > 0 && len(e.chunks[n-1].Data)+len(pkt) <= ChunkSize {
		e.chunks[n-1].Data = append(e.chunks[n-1].Data, pkt...)
		return nil
	}
	e.chunks = append(e.chunks, Chunk{Data: append(make([]byte, 0, ChunkSize), pkt...)})
	return nil
}

// Chunks returns the queued chunks without clearing them.
func (e *Encoder) Chunks() []Chunk { return e.chunks }

// Flush returns the concatenated stream and empties the queue.
func (e *Encoder) Flush() []byte {
	var out []byte
	for _, c := range e.chunks {
		out = append(out, c.Data...)
	}
	e.chunks = nil
	return out
}

// Compressor is the Huffman layer (FUN_004069c0 in Game.exe).
//
// TODO: the notes do not document the Huffman table, so no implementation is
// provided. Local/in-process mode skips compression entirely, which is all
// the offline game needs. Do not guess the table; extract it from the binary
// first.
type Compressor interface {
	Compress(dst, src []byte) []byte
	Decompress(dst, src []byte) ([]byte, error)
}

// Wire framing (SRV_SendPacketToClient, verified in game-net.md): each
// compressed blob is prefixed by its length in 1 byte when < 0xF0, otherwise
// 2 bytes whose first byte has high nibble 0xF.
//
// Unverified: that the 2-byte form is ((b0&0x0F)<<8)|b1 and that the length
// counts compressed bytes (the notes only state the nibble rule).

// AppendFrameLength appends the length prefix for a blob of n bytes.
func AppendFrameLength(dst []byte, n int) ([]byte, error) {
	switch {
	case n < 0 || n > 0xFFF:
		return dst, ErrTooLarge
	case n < 0xF0:
		return append(dst, byte(n)), nil
	}
	return append(dst, 0xF0|byte(n>>8), byte(n)), nil
}

// ReadFrameLength parses a length prefix. It returns the blob length and the
// prefix size; prefix == 0 means more bytes are needed.
func ReadFrameLength(b []byte) (n, prefix int) {
	if len(b) == 0 {
		return 0, 0
	}
	if b[0] < 0xF0 {
		return int(b[0]), 1
	}
	if len(b) < 2 {
		return 0, 0
	}
	return int(b[0]&0x0F)<<8 | int(b[1]), 2
}
