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
//   - Monster resists are applied per damage type (monstats ResDm.. columns)
//     plus the resist changes of curses (d2state): Amplify Damage doubles
//     physical damage taken. Stun and freeze hold a monster still, chill
//     slows it, fear (Terror, Howl) makes it run away from the hero, poison
//     and burn are damage over time (d2state streams ticked every frame).
//   - Auras, curses, buffs and charges are d2state sets per unit; the hero's
//     Stat() adds them and the passive skills (Dodge/Avoid/Evade roll through
//     the Director's HeroDefense hook, as do Energy Shield, Bone Armor and
//     Thorns). Summons are allied monsters of the Director (SpawnMinion);
//     sentries are stationary minions that fire the trap skill's missile.
//   - Hostile monsters target only Dopplezon and Valkyrie among the summons; summoned level is the owner's.
//   - Area missiles splash (hit functions 1 and 13) but have no special do
//     function movement (Blessed Hammer's spiral, Tornado's wander...).
//   - Weapon damage is the right hand weapon's min/max, or 1-2 bare handed;
//     ammunition comes from the game (Options.AmmoLeft / UseAmmo: the quiver
//     of a bow, or the stack of a thrown weapon; scenarios use Options.InfiniteAmmo).
package d2skills
