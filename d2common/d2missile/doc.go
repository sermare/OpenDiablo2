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
// walls stop a missile at once), the CollideType block masks (the cell is read
// through the mask, 0x6513e0: a wall bit in it stops the step and runs the hit
// function, a bit outside it is invisible to a moving missile), unit tests along the cells traversed in a frame,
// CollideFriend, LastCollide, the NextHit state shared on the target, pierce
// charges rolled at creation (up to 4, from skill_pierce + item_pierce), the
// ToHit roll, CollideKill and the hit functions 1 (area damage only), 2 and 4
// (sub missiles, damage and destroy), 10 (Guided Arrow with SrvDoFunc 7) and
// 14 (Meteor: area damage and 18 meteorfire). A hit function's return value
// REPLACES the result bits of ProcessHitOrExpire, so hit function 1 deals no
// direct damage; the hit function also runs on expiry, wall and path end.
//
// SrvDoFunc 2 and 6 spawn SubMissile1 for each subtile entered; 5 is an
// animation/footprint wobble without gameplay effect (not modelled).
// Collide type 1 also takes monsters with state 105 / stat 172 == 2.
//
// Not modelled: the remaining hit functions and SrvDoFuncs, the unit predicate
// of collide type 7 (missile versus CanDestroy missile), the town checks, periodic effects
// (20), the 64 direction quantisation of the path (the direction is exact
// here), the Explosion missile is client only (it has no pSrvDoFunc) and is
// reported as an event.
package d2missile
