// Package d2skill is the server side skill cast pipeline: it turns a skills.txt
// row plus a caster into mana payments, cooldowns, missiles, melee hits and
// state effects, the way SKILL_ServerRunStartFunc (0x56d4e0) and
// SKILL_ServerRunSkillFunc (0x56d6a0) do (skills-2.md, sections 1 to 5).
//
// The package is pure: the caster, the map and the engine are interfaces, the
// skill data are plain structs a loader fills in from skills.txt (see
// d2records.RecordManager.SkillTable), and every calc column is a compiled
// d2calc.Program. Missiles are created in a d2missile.Sim.
//
// Implemented skills (by srvdofunc / srvstfunc): the generic missile path
// (Fire Bolt, Ice Bolt, Magic Arrow, Fire Arrow, Cold Arrow, Fire Ball, Glacial
// Spike, Bone Spear, Holy Bolt, javelins...), Charged Bolt (do 17), Attack/Kick/
// Bash/Stun/Jab and the other plain melee skills (do 1, 2, 7), Throw (do 3),
// Frozen Armor and the other timed self states (do 18, 25, 47, 54, 116), Static
// Field (do 20), Inner Sight (do 6), the Nova rings (do 22) and Warmth
// (passive). Telekinesis and Find Item are not implemented.
//
// Class skills (class.go): a table maps each srvdofunc to a handler that reads
// the skills.txt columns, so skills sharing a function share code. Barbarian:
// Double Swing, Frenzy, Leap, Leap Attack, War Cry, Shout / Battle Orders, Taunt,
// Double Throw. Amazon: Multiple Shot, Guided Arrow, Charged Strike, Strafe,
// Fend, Lightning Strike. Necromancer: the curses (do 30, 59, 61), Teeth, Bone
// Armor, Poison Dagger, Corpse Explosion, Poison Explosion, Bone Wall / Prison,
// Raise Skeleton and the golems (summon orders). Paladin: Sacrifice, Smite, Zeal,
// Charge, Vengeance, the auras (do 65, 66, 81, 82). Sorceress: Chain Lightning,
// Fire Wall, Blaze, Energy Shield, Teleport, Meteor, Blizzard, Thunder Storm,
// Inferno. Druid: Firestorm, Twister, Tornado, Hurricane, Armageddon, Volcano,
// Raven, spirit wolves and totems, the forms, Maul / Feral Rage, Rabies.
// Assassin: the charge builders and finishers, Fire / Shock traps, sentries,
// Shadow Warrior, Psychic Hammer, Mind Blast.
//
// Outcomes that are not a missile or a melee strike come back as Effects
// (states, curses, auras, area hits, strikes, storms, summons, movement) for the
// engine to apply; d2state holds what a state means. Handlers cite the notes'
// function name and mark every behaviour that was inferred from the skills.txt
// columns rather than read from the binary with U. See Pipeline for what each
// does and what is unverified.
package d2skill
