package d2gs

// Packet is one framed game packet. Data includes the leading id byte.
type Packet struct {
	ID   byte
	Data []byte
}

// Decoder splits an already-decompressed byte stream into packets using the
// real size tables. It is not safe for concurrent use.
//
// Server->client streams are fully supported (fixed and variable lengths).
// Client->server streams are supported for the ids whose sizes are verified;
// any other id yields ErrUnknownSize because the stream cannot be resynced.
// After any error the decoder is poisoned (the stream is desynchronised)
// until Reset is called.
type Decoder struct {
	dir Direction
	buf []byte
	err error
}

// NewDecoder returns a decoder for the given direction.
func NewDecoder(dir Direction) *Decoder { return &Decoder{dir: dir} }

// Write appends stream bytes.
func (d *Decoder) Write(p []byte) { d.buf = append(d.buf, p...) }

// Buffered returns the number of bytes waiting to be framed.
func (d *Decoder) Buffered() int { return len(d.buf) }

// Reset drops buffered bytes and clears a sticky error.
func (d *Decoder) Reset() { d.buf, d.err = nil, nil }

// Next returns the next complete packet. ok is false when more bytes are
// needed (err nil) or the stream is broken (err non-nil).
func (d *Decoder) Next() (pkt Packet, ok bool, err error) {
	if d.err != nil {
		return Packet{}, false, d.err
	}
	if len(d.buf) == 0 {
		return Packet{}, false, nil
	}
	n, err := d.frameLen()
	if err != nil {
		d.err = err
		return Packet{}, false, err
	}
	if n == 0 || len(d.buf) < n {
		return Packet{}, false, nil
	}
	data := make([]byte, n)
	copy(data, d.buf[:n])
	d.buf = d.buf[n:]
	if len(d.buf) == 0 {
		d.buf = nil
	}
	return Packet{ID: data[0], Data: data}, true, nil
}

// frameLen returns the length of the packet at the head of the buffer, or 0
// when not yet determinable.
func (d *Decoder) frameLen() (int, error) {
	id := d.buf[0]
	if d.dir == ServerToClient {
		n, need, err := serverLength(d.buf)
		if err != nil {
			return 0, err
		}
		if need > 0 {
			return 0, nil
		}
		return n, nil
	}
	n, kind := ExpectedSize(id, ClientToServer)
	switch kind {
	case SizeFixed:
		return n, nil
	case SizeUnknown:
		return 0, ErrUnknownSize
	}
	return 0, ErrInvalidID
}

// SplitAll decodes a complete buffer into packets. Trailing partial data is
// returned as rest.
func SplitAll(dir Direction, b []byte) (pkts []Packet, rest []byte, err error) {
	d := NewDecoder(dir)
	d.Write(b)
	for {
		p, ok, err := d.Next()
		if err != nil {
			return pkts, d.buf, err
		}
		if !ok {
			return pkts, d.buf, nil
		}
		pkts = append(pkts, p)
	}
}
