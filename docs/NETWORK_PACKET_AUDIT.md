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
