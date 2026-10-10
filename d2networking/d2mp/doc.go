// Package d2mp is the multiplayer gameplay layer: an authoritative game
// simulation (Sim) and the replicas that follow it (Replica), with a compact
// event codec in between. It has no dependency on the engine, the renderer or
// game files, so a realm server can run it headless and tests can run several
// clients on the loopback.
//
// Division of labour:
//
//   - clients send what the player wants (walk/run/cast/select skill with the
//     verified C2S packets of package d2gs; interact, pick up and drop with the
//     d2gs packets whose layout is verified there; everything else, such as
//     party and trade commands, as Command messages in the realm tunnel);
//   - Sim decides what happens (movement, monster behaviour, missiles, damage,
//     death, drops, experience, level changes, party and trade) and emits
//     Events, in order, with the server clock;
//   - Replica applies the events of its level and answers "where is unit X at
//     this render time" with interpolation, so remote units glide instead of
//     teleporting and the local hero reacts at once (prediction).
//
// Movement is announced as segments (from, to, speed, start time), never as
// per-tick positions: every party computes Unit.PosAt from the same segment, so
// replicas agree with the server exactly and traffic stays small.
//
// Game rules (monster list of a level, damage, drops, walkable cells) come from
// a Rules value. DefaultRules is a self-contained, seeded rule set for headless
// use and tests; an engine host supplies its own from the game tables. All
// numbers in DefaultRules are placeholders, not Diablo II values (UNVERIFIED).
package d2mp
