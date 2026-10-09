// Package d2gsnet carries the engine's net packets over the real Diablo II game
// protocol framing (package d2gs): fixed-size packets, Huffman compressed
// blobs with a length prefix. It is selected with OD2_PROTO=d2gs and exists
// next to the JSON protocol, which stays the default.
//
// What is native D2GS: join (0x68, unverified layout), walk/run (0x01/0x03,
// verified), skill select and cast (0x3c, 0x05/0x0c verified), chat (0x15 /
// 0x26), leave (0x69 / 0x5c), game flags and act load (0x01 / 0x03 server ->
// client), player announce (0x59), unit movement (0x0f) and unit skill (0x4d).
// What has no counterpart (the hero's whole state, saves, waypoints) rides in
// a tunnel inside the variable meta packets 0xAE / 0x6c; see d2gs.Tunnel.
// Everything the receiving side does not understand is skipped by its size.
package d2gsnet
