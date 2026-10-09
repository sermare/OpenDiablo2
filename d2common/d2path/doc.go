// Package d2path holds map entity path types and a pure port of Diablo II's
// collision flags and path finder.
//
// The collision grid is a plane of 16-bit flags per subtile (collision.go).
// Routes are found the way the player's click-to-move path type 7 does
// (finder.go): a straight line first, then a short 8-neighbour A* with step
// costs 2 (straight) and 3 (diagonal) and heuristic 2*max+min. Everything here
// was taken from missiles-pathing.md; the file comments mark what is VERIFIED
// in the binary and what is an UNVERIFIED reading.
package d2path
