# boss-kill-speech (Game.exe 1.14b, read-only Ghidra; addresses and observations only)

(Written to docs/ because the worktree agent cannot write to ~/git/d2-re-notes; copy it there.)

Branch feat/boss-kill-speech (fork). Code: d2common/d2quest/boss_speech.go (+ test), hook in generic.go, scenarios in autoquest_acts2.go.

## Answer
The NPC lines after Mephisto / Diablo / Baal are NOT ambient "town cheers" sounds fired by the kill. They are normal quest speech lists
built when the hero clicks Talk (OnNpcInteract, S2C 0x27, voice via SOUND_LookupQuestMessageSound 0x4dd560 on open). Which table is sent
depends on per-hero record bits; hearing the message (OnNpcMessageHeard) flips a bit. The kill only sets bits (0x5b9d00, 0x5b2950, 0x58bae0).
Helpers: 0x65e820 get bit (rec, slot, bit), 0x65e870 set bit, 0x65e8b0 clear bit, 0x541480 (node, npc class, table index) appends the NPC's rows of
that speech table, 0x543130 membership in the node's "recently rewarded" list (count +0x9c, items +0x1c), added by 0x543070.

## A3Q6 Guardian (slot 0x16 = 22; OnNpcInteract 0x5b9e20, heard 0x5ba140)
- Interact: bit 0xb set -> table 5 (success lines, mode 1: Alkor 657, Ormus 658, Meshif 659, Asheara 660, Hratli 661, Cain 662, Natalya 663).
  Else if done (bit 0) and hero in the recently-rewarded list -> table 6 (same ids, mode 2 topics). Else (quest running) nIndices at 0x73e618.
- Heard (msg 0x291..0x297 = 657..663): bit 0xb set: if primary goal (0xd) set, timer 0x5b9dc0 + event 0x1e7 (not decoded); clear bit 0xb; add hero to list.
- Mephisto kill sets bit 0xb (see verify-bosses.md), so the cheers wait for the first Talk with any Kurast NPC. Any town NPC with a row, not only Alkor.

## A4Q2 Terror's End (slot 0x1a = 26; interact 0x5b23b0, heard 0x5b2290)
- Classic (game+0x70 == 0): bit 7 and Tyrael(367) -> table 2; bit 6 and Cain(246) -> table 2; bit 7 and Cain -> table 3; bit 6 and Tyrael -> table 3.
  Heard 684 (Tyrael) clears bit 7; 685 (Cain) clears bit 6. Bits 6, 7 are set by the classic Diablo kill.
- Expansion, quest done (bit 0): bit 9 clear: Tyrael -> table 4 (20000 mode 1), Cain -> table 5 (20001 topic); then bit 8 clear: Cain -> table 4, Tyrael -> table 5.
  Heard 20000 (0x4e20) sets bit 9 and, if a flag byte (+0x45 of node data) is clear, calls 0x5b2200 (spawns the quest object/NPC class 0x1cc: the portal to Harrogath, not decoded);
  heard 20001 sets bit 8. Not-yet-done branch uses nIndices at 0x73b23c (needs state < 4 or primary goal).

## A5Q6 Eve (slot 0x28 = 40; interact 0x58b7d0, heard 0x58b720)
- Done (bit 0) -> jump table 0x58b9d8 over class - 0x1ff. Tyrael(521): bit 0xd -> table 2 (20175), no per-NPC bit, so he repeats at every Talk.
  Larzuk(511) bit 4, Drehya(512) bit 9, Malah(513) bit 6, Qual-Kehk(515) bit 8: bit clear -> table 2 (their success line 20178/20176/20179/20180); bit set and 0xd -> table 3 (topic).
  Cain(520): bit 5 and bit 10 of the slot at node+0xe0, tables 4/5 (absent from the dumped csv): not decoded.
- Heard 20175..20180 sets bits 7, 9, 5, 4, 6, 8 (table at 0x58b7b0; 20175 -> 7, 20176 -> 9, 20177 -> 5, 20178 -> 4, 20179 -> 6, 20180 -> 8).
- Not-done: nIndices at 0x7333ec (state < 4).

## Unverified
Whether node+0xe0 is the quest's own slot; CanNpcTalk "!" markers while lines are pending (engine shows none); Cain in Harrogath; 0x5b2200 portal; timer 0x5b9dc0.
