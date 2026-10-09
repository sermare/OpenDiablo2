// Package d2daynight models the real game's global day/night environment
// (Game.exe 1.14b, Env.cpp; struct pointer DAT_007986bc), notes in
// d2-re-notes/renderer.md (b4), spot-checked in Ghidra read-only:
//
//   - phase 0..5, a tick counter, ticks-per-degree (+0x28) and ambient RGB.
//   - A day is 360 "degrees" of ticks-per-degree ticks each.
//   - The REAL outdoor sun cycle is the table at 0x740da0 (verified bytes):
//     12-byte entries {startDegree, lightIdx, R, G, B}: phase0 night blue
//     (7d 90 f3) from 320, phase1 dawn orange (d0 b8 83) from 340, phase2
//     white from 0, phase3 white from 160, phase4 dusk purple (c2 98 c1)
//     from 180, phase5 night blue from 200. Ticks per degree is 128 there, so
//     a day is 46080 ticks (30.7 min at 25 Hz, rate UNVERIFIED).
//   - The table at 0x740e30 (colours ~ (0,30,243), 4 ticks per degree) is the
//     special/scripted mode (the flag argument of SetPhaseAndTick); its
//     intensity is just 32. An earlier version of this package had the two
//     tables swapped.
//   - Ambient intensity (ENVIRON_UpdateAmbientIntensity 0x61b8a0, verified in
//     the decompile): a = tick/tpd degrees; I = 128+128*cos(a) while
//     tick < tpd*180, else 128+64*cos(a); values <= 0 become 0, capped at 255
//     (170 in Act 5). Level 0x78 is fixed at 200.
//   - The colour is blended from table[phase] towards table[phase+1] by the
//     progress through the phase: UNVERIFIED (x87 in the decompile).
//   - How the per-tick advance maps onto the phase counter is modelled here
//     as "phase is the last table entry whose start degree is <= the current
//     degree", not as the original incremental code (UNVERIFIED).
package d2daynight
