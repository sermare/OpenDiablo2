// Package d2sfx models the voice allocation rules of the original game's sound
// system (SoundHdr.cpp / the sound queue in Game.exe 1.14b) as a pure library:
// it knows nothing about Ebiten, the MPQs or wall-clock time. A Bank owns a
// fixed number of voices; callers hand it Sounds.txt rows, a Loader that makes
// Players, a Clock and a random source, so everything is testable with fakes.
//
// Provenance: behaviour marked "verified" was read from the decompiled binary
// (see ~/git/d2-re-notes/render-sound.md); "inferred" and "unverified" mark
// hypotheses.
package d2sfx
