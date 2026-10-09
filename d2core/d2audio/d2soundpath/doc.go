// Package d2soundpath maps the file names used by Sounds.txt and the engine to archive paths.
//
// Sounds.txt names are relative to one of several archive roots, depending on the kind of sound
// (verified against the 1.14b MPQs): sound effects live under data/global/sfx, NPC and monster
// speech (act1\akara\..., act1\monster\andarieltaunt1.wav) under data/local/sfx (d2speech, d2xtalk)
// and music (act1\town1.wav) under data/global/music (d2music, d2xmusic). Names use backslashes.
package d2soundpath
