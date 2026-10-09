// Package d2realm is a self-hosted realm/lobby for LAN and TCP/IP play. It
// talks to no Blizzard service. A Server hosts any number of games, a lobby
// with presence and chat, validates uploaded characters (.d2s) and persists
// them server side.
//
// The transport reuses the d2gs framing (Huffman blobs, d2gs.EncodeBlob and
// d2gs.ReadBlob) and the real packet ids where d2gs has them (chat, join,
// leave, player-in-game, game flags, load act). Everything the original
// kept on Battle.net (game list, game creation fields, character storage) is
// OUR extension and rides in the d2gs tunnel with the message types of
// protocol.go. See docs/NETWORK.md.
package d2realm
