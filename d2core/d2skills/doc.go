// Package d2skills runs the server-side skill pipeline (d2skill) and missile
// simulation (d2missile) inside the engine. It is the glue: the hero as a
// d2skill.Unit, monsters from d2monsters as missile targets, the map's DT1
// flags as the collision grid, and drawing entities that follow the simulated
// missiles.
//
// Engine level simplifications (everything else is in the pure packages):
//
//   - The cast runs locally (the hero and the monsters live in this process);
//     no packet is sent, so remote players do not see these casts.
//   - Skill levels come from the hero's skill points only (no +skills items).
//   - Mana is kept in 8.8 with a private fraction; there is no mana regen.
//   - Monster resists are applied per damage type (monstats ResDm.. columns);
//     stun, chill, freeze, poison and burn are not simulated, only logged.
//   - Howl's fear, Frozen Armor and Inner Sight are recorded as states (Inner
//     Sight really lowers monster defense for its duration); fear does not
//     make monsters flee.
//   - Weapon damage is the right hand weapon's min/max, or 1-2 bare handed;
//     the quiver is not modelled (arrow skills need OD2_AUTOCAST or the
//     Options.InfiniteAmmo switch to find "ammo").
package d2skills
