// Package d2party is the game's player roster: who is in the game, the
// parties, the invitations and the hostility between players, plus the rule
// that splits a kill's experience among a party. It is pure (no engine, no
// network): the server owns one Roster and broadcasts Snapshots, every client
// keeps a replica made with Restore.
//
// What the original does, from the binary (read-only, see the notes
// game-net.md and the Party.cpp server functions 0x53dca0..0x53e640):
//
//   - verified: a game keeps a list of parties (0x1d2c of the game object);
//     a party is an id plus a member list; removing a member from a party that
//     is left with fewer than 2 members frees the party (0x53e4e0).
//   - verified: the gold picked up and the experience of a kill reach the
//     party members that are in the same level (area) as the unit (0x53e2f0
//     iterates the members of the party whose level id equals the unit's;
//     0x53e640 divides picked-up gold by that count).
//   - verified: the roster the client draws (automap party dots, party panel)
//     holds name, class, level, party id and relation flags (Roster.cpp).
//
// UNVERIFIED (modelled after the player-visible behaviour): the invitation
// flow (invite, accept, decline), that hostility is a per-direction flag, that
// players cannot be hostile inside a party, and the level 9 requirement
// (taken from the existing party panel code, d2enum.PlayersHostileLevel).
package d2party
