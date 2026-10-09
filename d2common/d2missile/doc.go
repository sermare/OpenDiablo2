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
// Modelled (verified against the exe, see verify-missiles.md in the notes):
// lifetime in frames (Range + LevRange*level), velocity ((VelLev*lvl)/8 + Vel)
// << 8 with the 75/100 step scale, Accel/MaxVel (Accel is added to the scaled
// path velocity every 5th frame), the Activate collision delay (units only,
// walls stop a missile at once), the CollideType block masks (a wall bit in the
// mask stops the step and runs the hit function, a wall bit outside it ends
// the missile silently), unit tests along the cells traversed in a frame,
// CollideFriend, LastCollide, the NextHit state shared on the target, pierce
// charges rolled at creation (up to 4, from skill_pierce + item_pierce), the
// ToHit roll, CollideKill and the hit functions 1 (area damage only), 2 and 4
// (sub missiles, damage and destroy), 10 (Guided Arrow with SrvDoFunc 7) and
// 14 (Meteor: area damage and 18 meteorfire). A hit function's return value
// REPLACES the result bits of ProcessHitOrExpire, so hit function 1 deals no
// direct damage; the hit function also runs on expiry, wall and path end.
//
// Not modelled: the other pSrvDoFunc specials (2 and 6 spawn a SubMissile each
// frame the missile enters a new subtile, 5 is the meteorfire animation), the
// remaining hit functions, the unit predicates of collide types 1 (state 0x69
// monsters) and 7 (missile versus missile), the town checks, periodic effects
// (20), the 64 direction quantisation of the path (the direction is exact
// here), the Explosion missile is client only (it has no pSrvDoFunc) and is
// reported as an event.
package d2missile
