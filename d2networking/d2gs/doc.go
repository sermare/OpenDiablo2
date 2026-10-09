// Package d2gs implements the Diablo II 1.14b game server wire format as a pure library:
// the packet id and size tables (ids.go, size.go, tables.go), typed client and server messages
// (packets.go, server_msgs.go, moves.go), a streaming Decoder and Encoder with chunking, the
// game's Huffman coder (huffman.go), the blob framing used on TCP (wire.go) and a Tunnel for
// carrying engine state in the variable length meta packets. VERIFIED against the binary
// (read-only Ghidra, notes game-net.md): the size tables (server ids 0..0xB4, client ids
// 0..0x66) and the length functions, pinned by TestExpectedSizeServer, TestExpectedSizeClient
// and TestServerVariableLengths, plus decoder splitting, truncation and random-input safety
// tests. UNVERIFIED: the client to server blob framing (assumed equal to server to client) and
// the layout of some rarely used messages, each noted where it is defined. d2gsnet uses this
// package; the server and remote client select it with OD2_PROTO=d2gs while the JSON protocol
// of d2netpacket remains the default.
package d2gs
