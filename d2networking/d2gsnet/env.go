package d2gsnet

import (
	"errors"
	"io"
	"net"
	"os"
	"syscall"
)

// Enabled reports whether OD2_PROTO=d2gs selects the Diablo II game protocol
// for the TCP transport (server and client must agree; the default is JSON).
func Enabled() bool { return os.Getenv("OD2_PROTO") == "d2gs" }

// GameSeed picks the 32-bit game seed the protocol can carry: the map seed of
// the host's .d2s when it has one, else the low bits of the fallback seed.
func GameSeed(heroMapSeed uint32, fallback int64) int64 {
	if heroMapSeed != 0 {
		return int64(heroMapSeed)
	}

	return int64(uint32(fallback))
}

// IsClosed reports whether a read error just means the peer went away (EOF,
// reset, closed socket): a normal end of a connection, not a fault.
func IsClosed(err error) bool {
	return errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, net.ErrClosed) || errors.Is(err, syscall.ECONNRESET)
}
