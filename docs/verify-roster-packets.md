# verify-roster-packets: how the 1.14b client learns other players (Game.exe, read-only Ghidra)

Copy to ~/git/d2-re-notes/verify-roster-packets.md (the agent could not write outside its worktree).
Addresses and observations only. Roster entry = heap node (0xd8 alloc, list heads 0x7b3648 players / 0x7b364c free list; next ptr at +0x80). Lookup by unit id: 0x475000.

## Roster entry offsets (from the fill/getter functions)
| Off | Meaning | Evidence |
|---|---|---|
| +0x00 | name (NUL string, copied from packet) | 0x4750e0 |
| +0x10 | unit id | 0x4750e0 |
| +0x18 | s16 value from 0x65, getter named ROSTER_GetKillCount (0x475870) | 0x459df0 |
| +0x1c | class (dword) | 0x4750e0 |
| +0x20 | LEVEL (u16) | ROSTER_GetLevel 0x475880, used by the party-list row builder |
| +0x22 | PARTY ID (u16, 0xffff = none) | 0x8d worker 0x476020 writes it; ROSTER_GetPartyIdForUnit 0x4758f0 |
| +0x24 | level/area id (dword) | ROSTER_GetLevelId 0x475ed0 (setter not traced) |
| +0x28,+0x2c | x, y | ROSTER_GetPosition 0x475ea0 (setter not traced) |
| +0x30 | flags dword (getter ROSTER_GetPartyFlags 0x475a10, default 1 for null entry) | bit meanings UNVERIFIED |
| +0x38 | singly linked list of unit ids (8-byte nodes) | 0x8e |
| +0x44 | u16 from 0x5b @0x20 | meaning UNVERIFIED |
| +0x46 / +0x4a | account string / third string | 0x5b |
| +0x66 | display name: "name*account" via format at 0x6d5e0c when account non-empty, else name | 0x4750e0 |

## S2C packets
| Id | Size | Handler -> worker | Layout (verified unless noted) |
|---|---|---|---|
| 0x5b | var (u16 length @1; framing already in size.go) | 0x459c20 -> 0x4763d0 (add or update by unit id, regroup 0x4757a0, rebuild party list 0x496ac0) | u32 unit id @3, u8 class @7, name[16] @8, u16 level @0x18, u16 party @0x1a, u16 @0x1c IGNORED, u16 @0x1e -> flags +0x30, u16 @0x20 -> +0x44, NUL string account @0x22, NUL third string after it. 0x59 (AssignPlayer) is a position packet, so 0x5b is where level and party come from. |
| 0x75 | 13 | 0x459f30 -> 0x476540 | u32 unit id @1, u16 party @5 (+0x22), u16 level @7 (+0x20), u16 @9 ignored, u16 @0xb (+0x30). Updates an existing entry in place; then 0x474ce0, regroup, rebuild. |
| 0x8d | 7 | 0x45a1e0 -> 0x476020 | u32 unit id @1, u16 party id @5 -> +0x22; regroup/rebuild. |
| 0x65 | 7 | 0x459df0 | u32 unit id @1, s16 @5 (sign extended if bit 15 set) -> +0x18; regroup/rebuild. Not a "player kill event"; it updates a roster field whose getter is named kill count. |
| 0x8c | 11 | 0x45a1c0 -> 0x476060 | u32 A @1, u32 B @5, u16 flags @9. If A is in the roster: 0x4d90d0(A,B), 0x4d9270(A,B,flags) (pair relation table, not decoded), rebuild; if B equals the local player unit id, 0x474f60(A). Flag bits UNVERIFIED. |
| 0x8e | 10 | 0x45a1f0 -> 0x4760b0 (byte@1 != 0) / 0x476170 | u8 add @1, u32 A @2, u32 B @6. Adds/removes B in the list at A's entry +0x38 (creates A's entry if missing on add). Purpose UNVERIFIED. |
| 0x5a | 40 | 0x4597b0 -> UI_ShowEventMessage | copies the 40 bytes. u8 type @1 (switch 0..0x12; 0xc and 0xe fall through to nothing), u32 @3 (string id / object id for types 6, 0xf, 0x11), u8 @7 (type 6: 0 player, 1 monster, 2 object), name[16] @8 (byte 0x17 forced 0), 16 bytes @0x18 (name, or u16 super-unique id for type 6 kind 1). @2 not read in traced paths. Types 2 and 3 compare the subject name to the local player (type 2 skips when equal). Type 7 goes to UI_FormatPartyEventText (not decoded). Mapping of type to message text UNVERIFIED (string ids not resolved). |
| 0x67-0x6a | 16/21/12/12 | not roster: control ids (create/join/leave game) per game-net.md; not decoded further here. |

## Gaps
- Level-up of a present player: no dedicated packet found; 0x75 carries level and party, probably the mechanism (UNVERIFIED which server events send it).
- 0x7a (13), 0x7f (10), 0x90 (13), 0x8b (6), 0xab not decoded; 0x7a (0x459fb0) is a party add/remove keyed on byte@1 (calls 0x474850 / 0x4749d0).

## Code
d2networking/d2gs/roster_msgs.go (+ roster_test.go: hand-built byte fixtures, FuzzRosterParsers), d2networking/d2gsnet/translate.go (server emits 0x5b with AddPlayer and a 0x75 per roster member; client stores PeerInfo and lets the verified level/party win over the tunnelled copies). The tunnelled AddPlayer / RosterUpdate remain the OD2 extension for hero state, hostility, invitations, areas and notices.
