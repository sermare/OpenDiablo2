package d2gs

import (
	"bufio"
	"fmt"
	"io"
)

// Blob framing for the TCP transport: a blob is a run of whole packets,
// Huffman compressed, prefixed by its compressed length with AppendFrameLength
// (1 byte if < 0xF0, else 2 bytes with high nibble 0xF; the original's server
// -> client framing, game-net.md). Unverified: the original's client -> server
// framing is undocumented, so both directions use this one; and the length
// counts compressed bytes.

var defaultHuffman = NewHuffman()

// EncodeBlob frames and compresses packets into one wire blob. The packets
// must fit the 0xFFF length of the prefix once compressed.
func EncodeBlob(packets ...[]byte) ([]byte, error) {
	var plain []byte

	for _, p := range packets {
		plain = append(plain, p...)
	}

	comp := defaultHuffman.Compress(nil, plain)

	out, err := AppendFrameLength(nil, len(comp))
	if err != nil {
		return nil, err
	}

	return append(out, comp...), nil
}

// maxBlobPlain bounds the plain bytes of one blob so that the compressed form
// (at most 11/8 of it) stays under the 0xFFF limit of the length prefix.
const maxBlobPlain = 0x800

// EncodeStream frames packets into as many blobs as needed (each blob holds
// whole packets) and returns them concatenated.
func EncodeStream(packets ...[]byte) ([]byte, error) {
	var (
		out   []byte
		batch [][]byte
		size  int
	)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}

		b, err := EncodeBlob(batch...)
		if err != nil {
			return err
		}

		out, batch, size = append(out, b...), nil, 0

		return nil
	}

	for _, p := range packets {
		if size+len(p) > maxBlobPlain {
			if err := flush(); err != nil {
				return nil, err
			}
		}

		batch, size = append(batch, p), size+len(p)
	}

	return out, flush()
}

// ReadBlob reads one blob from r and returns the decompressed packet bytes.
func ReadBlob(r *bufio.Reader) ([]byte, error) {
	first, err := r.ReadByte()
	if err != nil {
		return nil, err
	}

	n := int(first)

	if first >= 0xF0 {
		second, err := r.ReadByte()
		if err != nil {
			return nil, err
		}

		n = int(first&0x0F)<<8 | int(second)
	}

	comp := make([]byte, n)
	if _, err := io.ReadFull(r, comp); err != nil {
		return nil, err
	}

	return defaultHuffman.Decompress(nil, comp)
}

// ClientPacketLength returns the length of the client packet at the head of b
// (0 when b is too short to tell). Fixed sizes come from the verified table;
// 0x15 chat and the engine's control packets (0x68 join, 0x69 leave, 0x6c
// tunnel) have their own rules. Any other id returns an error: its size is not
// known, so the rest of the blob cannot be framed and is ignored by callers.
func ClientPacketLength(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	switch b[0] {
	case C2SChat:
		n := strlen(b, 3)
		if len(b) < 4 || n < 0 {
			return 0, nil
		}

		// message NUL, then the recipient name NUL
		m := strlen(b, 3+n+1)
		if m < 0 {
			return 0, nil
		}

		return 3 + n + 1 + m + 1, nil
	case CtlJoinGame:
		return joinGameSize, nil
	case CtlLeaveGame:
		return len(LeaveGame()), nil
	case CtlTunnel:
		if len(b) < 3 {
			return 0, nil
		}

		return 3 + (int(b[1]) | int(b[2])<<8), nil
	}

	n, kind := ExpectedSize(b[0], ClientToServer)
	if kind != SizeFixed {
		return 0, fmt.Errorf("%w: client id %#x", ErrUnknownSize, b[0])
	}

	return n, nil
}

// SplitClient splits a decompressed client blob into packets. A packet of an
// unknown id ends the split; ignored is the number of bytes dropped from there.
func SplitClient(b []byte) (pkts [][]byte, ignored int) {
	for len(b) > 0 {
		n, err := ClientPacketLength(b)
		if err != nil || n == 0 || n > len(b) || n > MaxPacketSize {
			return pkts, len(b)
		}

		pkts = append(pkts, b[:n])
		b = b[n:]
	}

	return pkts, 0
}

// SplitServer splits a decompressed server blob into packets using the
// verified size table; on a framing error the rest is dropped (ignored).
func SplitServer(b []byte) (pkts [][]byte, ignored int) {
	pk, rest, err := SplitAll(ServerToClient, b)
	for _, p := range pk {
		pkts = append(pkts, p.Data)
	}

	if err != nil || len(rest) > 0 {
		return pkts, len(rest)
	}

	return pkts, 0
}
