// Package d2object holds the rules of the world objects that are not doors,
// waypoints or portals: what each objects.txt OperateFn does (as data), the
// shrine effects with their durations and stat changes, the wells and the
// timed effect list that puts shrine stats on the hero.
//
// The package is pure (no engine, no MPQ access). The engine glue lives in
// d2game/d2gamescreen/objects_operate.go.
//
// Sources. VERIFIED in Game.exe 1.14b: the server dispatches an object click
// through a table of function pointers at 0x730258 indexed by the objects.txt
// OperateFn column (entry 15 = SERVER_UsePortalObject 0x582700, 23 = the
// waypoint function 0x582d00, 2 = the shrine function 0x581b00, 20 = weapon
// rack 0x582050); the shrine function runs only when the object is in mode 0
// and its "used" word (object data +0xc) is 0, stores the operating player,
// sets mode 1 (Operating), calls the effect function of the shrine type
// (table at 0x6e2b48, 12 bytes per type, 0..22; the shrine type is object
// data byte +4 and the message string is 0xe63 + type) and schedules the
// reset event after ResetMinutes*0x4b0+1 frames. The resist/armor/mana
// recharge/stamina/experience shrines apply a STATE_SHRINE_* state (states
// 128..137) carrying one stat each (39/43/41/45 resists, 171
// skill_armor_percent, 27 manarecoverybonus, 162 skill_staminapercent, 85
// experience); the weapon rack function sets mode 2 and clears the selectable
// flag after dropping its items. Everything else below is marked UNVERIFIED.
package d2object
