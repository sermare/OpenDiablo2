// Package ebiten is the Ebiten backed audio provider: it loads sounds through the asset
// manager, decodes WAV data, plays it through Ebiten's audio context and owns the volumes and
// panning of effects and music. The voice allocation rules live in the pure d2sfx package; this
// one only talks to the sound device, so it cannot be unit tested without one. Upstream code,
// adapted by the fork. Not compared against the original.
package ebiten
