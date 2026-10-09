// Package d2lightmap implements the real game's 48x48 subtile light map
// (Game.exe 1.14b, LIGHT_RebuildLightMap 0x4714e0 and friends; notes in
// d2-re-notes/renderer.md section b4). The map is rebuilt once per game tick
// (25 Hz): cleared to the base light of the level, then every dynamic light
// source paints a radial falloff into it.
//
// Verified in the notes: 48x48 cells of one subtile, origin = hero subtile
// minus 24, positions with 3 fractional bits, octagon distance
// (max*0x3d7 + min*0x197)>>10, linear falloff painter, intensity-weighted
// colour blend in AddToCell, radius ramp of 8 units per tick capped at 0xf8.
// Not implemented: the line-of-sight painter used at the highest quality
// setting (walls do not block light) and the 0x470250 tinting of other levels.
// The reciprocal table at 0x7a8af0 is assumed to be 65536/n (UNVERIFIED).
package d2lightmap
