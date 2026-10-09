// Package d2quest implements the quest system of the original game as a pure,
// engine independent state machine: a list of quest nodes, a set of events
// the engine feeds in (area entered, monster killed, item picked up, NPC
// talked to, message heard...), the 16 flag bits of the d2s quest record
// they drive, and the effects (skill point, mercenary, imbue, items...) the
// engine has to apply.
//
// The model follows the 1.14b binary notes (d2-re-notes/quests.md and
// quests-2.md) and the clean-room D2MOO sources the notes compare against.
// Differences to the real game, all deliberate:
//
//   - one player per game: the party loops (members in the same room, party
//     members elsewhere in the act) collapse to "the hero"; the per-player
//     GUID lists of the original become booleans on the quest;
//   - object, missile and portal effects are reported as Effects instead of
//     being simulated;
//   - the quests of Acts 2-5 are implemented from the notes (Act 2 in detail,
//     Acts 3-5 as table driven nodes whose triggers are UNVERIFIED, see
//     generic.go); the "act finished" words and the unlock of the next act are
//     set by the TravelToActN calls.
//
// Evidence: the state machines of Act 1 and Radament's Lair follow the clean-room
// D2MOO quest sources, which the notes checked against the 1.14b binary for the
// speech tables (all tables, byte for byte) and for the A1Q1/A1Q2/A2Q1 message
// handlers. UNVERIFIED against the binary: the other quests' handlers, the 1.14b
// only deltas (the Akara respec bits are from the binary notes, the Uber quests
// are not modelled), the ids of the attach-sound effects, the topic captions of
// the Talk submenu, the heard-list handling of the client, the barking distance,
// and the Charsi message 150 mode (the binary dump says topic, D2MOO says spoken;
// the binary table is used).
//
// Bit meanings are in d2s.QuestBit*; see the constants below for the names
// the quest code uses.
package d2quest
