// Package d2missile simulates server-side missiles: creation from a
// missiles.txt row, the per-frame standard move (MISSILE_SrvDoStandardMove,
// 0x5abd00), the collision test along the path and the hit/expire processing
// (MISSILE_ProcessHitOrExpire, 0x5aba10), see missiles-pathing.md and
// skills-combat.md (e) in the notes.
//
// The package is pure: the map, the units and the damage sink are behind the
// small World interface, and the missile data are plain structs a caller
// builds from missiles.txt.
//
// Modelled (verified in the notes unless marked): lifetime in frames
// (Range + LevRange*level), velocity ((VelLev*lvl)/8 + Vel) << 8 with the
// 75/100 step scale, Accel/MaxVel (unit of Accel UNVERIFIED), the Activate
// collision delay, wall tests with the CollideType masks, unit tests along the
// cells traversed in a frame, CollideFriend, LastCollide, NextHit/NextDelay,
// Pierce charges, the ToHit roll, CollideKill, AlwaysExplode and the
// HitSubMissile spawning of hit functions 2 and 4.
//
// Not modelled: homing/SpecialSetup, the per-missile pSrvDoFunc specials
// (only the standard move and SubMissile spawning of do func 2), area hit
// functions (1, 24), periodic effects (20), the 64 direction quantisation of
// the path (the direction is exact here), the Explosion missile is client
// only (it has no pSrvDoFunc) and is reported as an event.
package d2missile
