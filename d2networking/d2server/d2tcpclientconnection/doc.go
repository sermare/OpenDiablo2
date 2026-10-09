// Package d2tcpclientconnection is the TCP implementation of the server side client
// connection: it reads and writes the net packets of one client on a TCP socket and feeds them
// to the GameServer (game_server.go creates it when a client joins). Upstream code with no unit
// tests of its own; the multiplayer scenario scripts/verify.d/96-multiplayer.sh exercises it.
// Nothing here is compared against the original protocol (that is d2gs and d2gsnet).
package d2tcpclientconnection
