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
package d2monster
