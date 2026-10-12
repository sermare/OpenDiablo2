# Multiplayer packet layer audit (d2networking/d2gs vs Game.exe 1.14b)

Method: read-only Ghidra (port 8089). The S2C length table (dword table 0x72e958, ids 0..0xB4)
and the 12-byte client handler table at 0x710f70 ({handler, size, alt handler}) were read back and
compared id by id with `serverSizes`; the C2S dispatch table at 0x6e2010 (8-byte entries) was read
back and compared with `clientHandled`. Handler bodies were disassembled for field offsets.
Facts only, no game code. Pinned by `d2networking/d2gs/exe_pin_test.go`.

## Result

- S2C sizes: all 181 entries (ids 0..0xB4) of the length table match the port, fixed sizes and the
  variable markers. No wire-size mismatch.
- C2S handled set: null handlers at 0x2b, 0x4a, 0x4e, 0x55-0x57, 0x5a-0x5c, 0x64, 0x65 (and 0) match
  `clientHandled` exactly.
- Handler table vs length table, two oddities the port follows from the length table:
  - 0x17 (client name `UnitActionNonLocalPlayer`, variable) has a handler but length 0, so the
    framing function rejects it; no sender was found in the notes. Port: invalid.
  - 0x80 has size 4 in the handler table but a NULL handler and length 0. Port: invalid.
- The Huffman table and the blob framing were not re-read in this pass (the earlier verified-by-oracle
  tests in huffman/wire stay as they are). The C2S blob framing remains the unverified assumption
  noted in doc.go.

## Wire-compat fixes made (layouts read from handlers and senders)

- S2C 0x01 GameFlags: handler 0x458020 reads byte @1, dword @2, byte @6, byte @7. The port wrote
  expansion at 5 and ladder at 6; now expansion @6 and ladder @7, hardcore in the @4 word. Which of
  @6/@7 is expansion vs ladder follows the public layout (the handler only stores both in globals).
- S2C 0x4d unit skill at location (17): handler 0x4596a0 and sender 0x53b1e0 agree on dword @6 (skill
  id in the low word), byte @10, u16 @11, u16 @13. The port had skill u16 @6, level @8, x @9, y @11.
  Now fixed; the byte at @10 is called Level (meaning inferred).
- S2C 0x0f PlayerMove (handler 0x4584b0), 0x03 LoadAct (0x458050): already matched.
- Names: 0x4c UnitSkillOnTarget, 0x4d UnitSkillAtLocation, 0x67-0x6d NPC move/state/attack/stop
  (0x67 inferred), 0x17 added to the name table.

## Packets the original sends that the port has no typed message for

The decoder still frames and skips all of them by size. Typed messages exist only for S2C ids 00-06,
0f, 18, 26, 4d, 59, 5a, 5b, 5c, 65, 75, 8b-8e, 95, 96, 9c/9d (framing), ae, af, b2-b4. Not typed (selection):

- Unit actions: 0c UnitActionBasic (9), 0d unit stop (13), 0e object state (12), 10 move to entity
  (16), 17 non-local unit action (variable), 15 reassign unit (11), 4c skill on target (16), 99/9a
  copies of 4c/4d for the local unit (16/17).
- Monster movement and combat: 67 move to location (16), 68 move to target (21), 69/6a state change
  (12), 6b/6c attack location/target (16), 6d stop (10). Note 6b is a distinct handler (position fix).
- Stats and skills: 19-1f add/set attribute (sizes 2,2,3,5,3,4,6; server picks 1d/1e/1f by value
  width), 20 set player stat (10), 21 update skill level (12), 23 set active skill (13), 7b skill slot (8).
- World and session: 09 assign level warp (11), 29 quest records (97), 63 waypoint menu (21), 61 play
  cinematic (2), 73 missile create (32), 7e reload act COF cache (5), 79 trade action (6), 81 assign
  merc (20), 91 NPC greeting list (26), a3 class event (24), 5e/52 38- and 42-byte copied packets.
- Id 0x80: size 4 in the handler table, NULL handler (never delivered).

## C2S

Only ids with a verified handler length are in `clientSizes`; the rest of 1..0x66 are handled by the
exe but report SizeUnknown (not framed). Remaining unverified C2S sizes: 0x14, 0x15 (variable chat),
0x2b-0x2e area, 0x39-0x57 except those listed, 0x60-0x66. Typed C2S messages cover movement, casting,
skill select, chat, item/trade/NPC ids by number only (no field structs for the item moves).

## Open items

- Verify the remaining C2S lengths from handler bodies (0x543xxx-0x55xxxx range) and extend `clientSizes`.
- Type the unit-action family (0c/0d/0e/10/4c/67-6d) so remote monsters can be mirrored natively.
- Re-read the Huffman frequency table against the exe and pin it in a test.

## w48 pass: typed unit-action, attribute, skill and world packets

Layouts read read-only from the client handlers (Ghidra, handler addresses in the Go comments) and
the server senders 0x538ff0 (0d), 0x53b250 (4c), 0x539a70 (1d-1f), 0x539e20 (20), 0x53b720 +
0x582d00 (63), 0x53b7d0 + 0x59daa0 (73). New typed messages, all with round-trip tests that also
check the exe length table (`d2gs/unit_action_msgs.go`, `stat_world_msgs.go`, `DecodeServer`):
S2C 0c, 0d, 0e, 10, 4c (+99), 67-6d, 19-1f (`AttrUpdate`: 19/1a/1b add gold/exp, 1c sets exp,
1d-1f set a stat; width chosen by the sender at <0xff, <0xffff), 20, 21, 23, 7b, 63, 73.
Byte-sized fields whose meaning was not decoded are carried verbatim and marked UNVERIFIED.

Corrections to earlier notes: 0x4c has the target type at @9 and target id at @10 (not a skill
location); 0x68 and 0x6b/0x6c apply a position correction (0x47c320) before the action; 0x6b = 0x69
layout + correction at @12/@14, 0x6c = 0x6a layout + correction; 0x6d has no mode byte.

Client -> server lengths verified from the handlers (each rejects other lengths with code 3) and
added to `clientSizes` with typed messages (`client_msgs3.go`): 0b:1, 3d:5, 3e:5, 3f:3, 40:1,
41:1, 44:17, 46:13, 47:13, 48:1, 49:9 (take waypoint), 4b:9, 4c:5, 4d:3, 4f:7, 50:9, 53:1, 54:1,
60:1, 61:3, 62:5, 63:5. Handled ids whose handler ignores or always rejects any length (so no
length exists to read): 2c, 2d, 2e, 39, 42, 43, 45, 52, 66; they stay SizeUnknown. 0x14/0x15 are variable.

Port multiplayer wiring (decided per packet; nothing new is sent or consumed on the wire, so the
9j/96/9d scenarios are unchanged):
- Attributes, skill level/slot/active skill: the port syncs hero state through the tunnelled
  AddPlayer/Save packets; no engine packet carries single stat changes, so emitting 19-23/7b would
  have no consumer.
- Waypoint menu 63 / take waypoint 49: the port's waypoint flow is the engine SetWaypoint packet
  (activation), the menu is local UI; no equivalent exists to translate.
- Missile create 73: missiles are simulated locally by each client from the cast packet (0x4d); 73
  would double-spawn them.
- Unit actions 0c-10, 4c, 67-6d: the monster simulation travels as RealmUnit (appear/move/hit/die
  with OD2 fields these packets cannot carry). Mapping move/stop to 67/6d is possible later but needs
  an id table for realm monsters and a client consumer; left out to keep the scenarios stable.
