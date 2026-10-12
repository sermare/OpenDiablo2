# w49 multiplayer items follow-ups (branch feat/w49-mp-items2, base fork/land/w45)

Own words, addresses only. Ghidra D2_1_14b read-only.

## Exe facts (VERIFIED this wave)
- ITEMMODE_ServerHandlePickupRequest 0x55ace0 takes only (game, player, unit id, out flag): no distance test, no owner/timer test.
  Order: empty cursor, not interaction-busy, unit found by (4,id), mode must be 3, then gate 0x55aa20. Distance is a client-walk matter.
- 0x55aa20 ITEMMODE_CanPickupQuestOrUniqueItem: (1) unique whose UniqueItems row has bit 4 at +0x2c (carry1) and conflict test
  0x55a7d0 true -> refuse; (2) quest byte 0 -> allow; (3) quest items: per code the quest flags of the player's difficulty record
  (codes "ass ", "j34 ", "xyz ", "g33 ", "tr2 " each test one or two flags), then per quest byte 9 / 4 / a table at 0x72f8b8 a flag must
  be clear; (4) the conflict test again; (5) refuses when a corpse of the player holds a conflicting item.
- 0x55a7d0 conflict: scans the player's items (itself excluded), skipping items whose inv page byte is 1; same carry1 unique file
  index, or same quest byte with the same code or a paired code (j34/g34, bks/bkd, d33/g33, hst/msf, hst/vip, qf1/qf2, qf2 with
  qhr/qey/qbr). Which container is "page 1" is UNVERIFIED (the port treats it as worn/belt and skips it).
- Visibility: SERVER_SendUnitIfRoomNearClient 0x52b9f0 sends a unit to a client only when the unit's room is in the near-room array
  of the client's current room (ROOM_GetNearRoomArrayAndCount; ROOM_IsNearRoomOfUnit 0x536240 is the same membership test).
  So a drop is seen by heroes in the drop's room or an adjacent one, not by the whole level. The port keeps showing drops to the
  whole level (a superset, with a sync on level entry); a room filter would also need a re-sync when a hero walks into range.
- Loot ownership: none (free for all), as w24.

## Implemented
- d2ground.CanPickUp/Conflicts (pure, tested), GameControls.CanPickUpGround; Game.pickUp refuses before any server request.
  Quest-flag half and the corpse scan are not ported (quest state is client owned).
- Single-roll loot: ops claim/grant/lost. Monster (Director.LootGate/OnLoot) and chest loot ask the server for the roll by key;
  the first claimant rolls and reports the drop (broadcast to the level), the others wait for the broadcast. The server forgets the
  claims of a level when no hero is in it. Monster loot was not announced to the server at all before this change.
- Scenario 9j-loot-claim; 9j-ground-items and 96-multiplayer still pass.

## Open
Monster brain ids match across clients only if both spawn the level alike; a chest opened by one hero stays closed for the other
(no object sync). Ears are normal announced items; gold overflow and death piles already announce with an owner.
