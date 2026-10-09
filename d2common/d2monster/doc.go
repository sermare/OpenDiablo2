// Package d2monster is a pure port of Diablo II's monster AI framework.
//
// It was written from the reverse engineering notes monster-ai.md and
// monster-ai-2.md (Game.exe 1.14b). Nothing here touches the engine, the
// renderer or game files: a monster is a Brain (per-unit RNG, AI scratch
// state, command queue, leader/minion links) and the outside world is reached
// through three small interfaces (Senses, Actor, World).
//
// Faithfulness convention used in comments:
//
//	VERIFIED   - control flow and constants read from the decompiled think
//	             function (see the notes).
//	UNVERIFIED - the notes mark it so, or it is an interpretation of a vague
//	             note. Such parts are kept minimal and replaceable.
//
// The think functions never move or attack directly; like the original they
// request one action (attack, move, cast) or a sleep, and the engine runs the
// next think when that action finishes (Brain.WakeNow) or the sleep expires.
// RNG consumption order matches the original: every roll advances the
// per-monster seed once and && short-circuits skip rolls.
//
// Ported think functions: Idle, None, Skeleton, Goatman, Swarm, Wraith, Zombie,
// Brute (with the observed aip3-twice quirk), Mummy, Scarab, Bighead,
// CorruptRogue, Fallen, SkeletonBow, CorruptArcher, SkeletonMage,
// PantherJavelin, Andariel, Smith, Griswold and BloodRaven (earlier passes);
// Vulture, Summoner (slot mapping now verified against the decompilation),
// Duriel, Mephisto, Diablo, Izual, BaalMinion and SuicideMinion (ai_boss2.go).
//
// Forced states (forced.go): the alternate AI table at 0x73a548 (states 2, 3,
// 6, 8, 9, 10, 11, 12, 14, 17) and the Fear / Blind / Taunt / Confuse /
// Attract / Charm conditions with their durations, their override of the think
// function and the restore of the class AI.
//
// Ported in the boss pass (ai_baal.go, ai_shaman.go): FallenShaman, the Baal
// wave AIs BaalThrone, BaalTaunt, BaalToStairs, BaalCrab(+Clone), BaalTentacle
// and the forced states 13 and 16.
//
// Not ported: PantherWoman, QuillRat,
// SandLeaper, SandRaider, Fetish, CorruptLancer, GreaterMummy, the monster-side hooks of
// Cloak of Shadows / Overseer whip, and MONAI_PostTargetChecks (wounded
// MonTeleport, Summoner wake-up, threat re-targeting).
package d2monster
