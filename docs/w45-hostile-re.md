# w45 hostility toggle, town rules, loot, pets (Game.exe 1.14b, Ghidra read-only; addresses and observations only)

## Hostility toggle 0x5a3870 resolved (VERIFIED)
- Reached through the table at 0x6e36c8 (entry index 4 = 0x6e36d8), called by PARTY_ProcessPartyAction 0x5a3a10 as handler(unit, targetId, &err).
  Callers: SERVER_HandleC2S5D_SetPlayerRelation 0x54a0c0 and SERVER_HandleC2S5E_PartyRequest. 5D requires ARENA_IsFlag10000Set()==0 else returns 3,
  and action index > 10 returns 2.
- Entries: 1 0x5a31d0 (not a defined function, probably flag 1 = loot), 2 0x5a32a0 flag 2, 3 0x5a3300 flag 4, 4 0x5a3870 flag 8 (hostile),
  6 0x5a3640, 7 0x5a3590, 8 0x5a3490 (form party), 9 0x5a3760.
- Gate: declarer room non-null, ROOM_IsTownRoom (0x61a850 -> LEVEL_IsTownLevelId 0x643b50: levels 1, 40, 75, 103, 109) TRUE, declarer base level (stat 0xc) >= 9
  and target base level >= 9. Any failure: *err = 1. PARTY_ProcessPartyAction then returns 3, the same value CCMD_DispatchClientPacket returns for an invalid
  packet id, i.e. a protocol violation; no message is shown. The stock client only offers the hostile button in town. So hostility is declared AND withdrawn in
  town (peace goes through the same gate) and fights happen in the field. The previous port polarity (refuse in town) was wrong.
- Peace path: only if the other player's non-base level >= 9 or the flag was set. Declarations obey the 60 s cooldown (GetTickCount + 60000); during the
  cooldown the declarer gets unit mode 0x13 and a trade-status packet instead.

## Damage in town (VERIFIED)
COMBAT_ApplyDamageToUnit 0x57a640: if UNIT_AreUnitsHostile is 0 (and no flag 0x1000) the damage bits are cleared; then if the VICTIM's room is a town room the hit is
cancelled unless the attacker is a monster with monstats bitfield 0x400. So the victim's room decides (the old port tested the attacker only; the client test
0x461460 keeps testing the attacker). Port: either end in town refuses.

## Pets and mercenaries (VERIFIED)
UNIT_AreUnitsHostile 0x552270 maps unit A (monster -> MONAI_GetLeaderUnit; missile with flagsEx bit 10 -> its owner) and follows unit B's leader chain; the same
resolved unit -> not hostile; two players -> relation flag 8 of A toward B; otherwise the monster relation (UNIT_CompareStateStatACRelation). A hero's pet or merc
attacks, and is attackable by, another hero exactly when the owners are hostile (MONAI scans and missile hits alike).

## Corpse looting (VERIFIED)
PLAYER_CanLootCorpse 0x57db10: owner (INV_GetOwnerId of the corpse inventory) == looter, or relation flag 1 of owner toward looter; otherwise the loot request is
dropped (PLAYER_ServerLootCorpse 0x57db90, which also needs state 7 on the corpse). The owner looting also restores stat 0xd. Port: d2party Roster.SetLootAllowed
and CanLootCorpse (roster API only; the port has no corpse object to hook it to yet). That flag 1 is what the "allow loot" check box sends is unverified.

## Quest kills in a party (VERIFIED for Den of Evil only)
QUEST_A1_DenOfEvil_OnMonsterKilled 0x58e010 counts the cleared monsters of the level region, not the killer. On completion it sets the quest state and runs
SERVER_FindUnitInHashByPredicate over ALL players of the game with 0x58de30 (completion flags for each player whose record is not complete), plus
QUEST_GrantFlagToPlayersInList. Quest kill credit is therefore game-wide, not limited to killer or party. Other quests not traced; the port does not model it.
