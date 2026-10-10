# verify-item-rules: item rules checked in Game.exe 1.14b (addresses and observations only)

(Written to docs/ because the agent may not write outside its worktree; copy to ~/git/d2-re-notes/verify-item-rules.md.)

Type ids: ItemTypes ids are the row index from 0, but the loader skips the "Expansion" marker row (line 60 of ItemTypes.txt), so ids after it are one lower than the line index. Confirmed: 0x12 Book, 0x25 Helm, 0x2d Weapon, 0x32 Any Armor, 0x43 Hand to Hand (h2h, claws), 3 Armor (tors).

## 1. 0x62bd70 ITEM_GetMaxSocketsForLevel - VERIFIED
- Result = min(base record byte +0x138 (gemsockets), ItemTypes row byte +0x1a / +0x1b / +0x1c = MaxSock1 / MaxSock25 / MaxSock40). Item level (0x628480, min 1): ilvl <= 25 -> MaxSock1, ilvl <= 40 -> MaxSock25, else MaxSock40. The old Go border (ilvl < 40) was wrong at ilvl 40.
- Plate Mail gemsockets 2 is the armor.txt base column (tors MaxSock columns are 3/4/6); the cap is the min of both.
- 0x554c60 RollSockets: quality >= 2, base hasinv, not stackable, max > 0. Difficulty cap 3/4/6 (game+0x6d). rand(100) < 33 (request flag 0x10 forces roll 0, 0x8 blocks, item flag 0x800 forces success). Format != 0: count = initSeed (pItemData +0x10) % cap + 1; format 0 (classic): min(cap, 3), helm (type 0x25) at most 2; format < 100 body armor (type 3) never rolls. Then 0x62be00 ClampSocketCountBySize: area = invwidth*invheight (base +0x10f, +0x110) capped to 6, min with level max, min with requested (>= 1), stored in stat 0xc2, flag 0x800 set.

## 2. Requirements - VERIFIED
- 0x62ebf0: bonus = MulDiv(base, stat 0x5b, 100) (0x47f2c0 = a*b/c, truncating signed) for str (+0x10a) and dex (+0x10c); ethereal (item flag 0x400000) subtracts 10 from the BONUS, then base is added. Stat check needs hero stat >= 1. A Book (type 0x12) needs quantity stat 0x46 > 0; class restriction 7 = any, else equal to hero class; the item must be identified (flag 0x10).
- 0x62b720 level, by quality: magic = highest of prefix/suffix/automagic affix level (affix record +0x65, or +0x68 when record +0x67 equals a class arg); set = SetItems +0x32; rare = highest of the six affix rows and the name row; unique = UniqueItems +0x36; crafted = highest affix + 10 + 3 per affix, capped to max hero level - 1. Floor by base levelreq (+0x13f); max over socketed items; skills from stat 0x6b (Skills reqlevel +0x174) and 0x61 (that + 6 unless class matches); finally ADD stat 0x5c; 0 if below 1. The equip check compares hero level (stat 0xc) with it.

## 3. Durability loss chance - VERIFIED (code was right, itemgen.md had it reversed)
0x557d90: armor (type 0x32) 10%, weapon (type 0x2d) 4%; throwable weapon 10% in an expansion game (game+0x70), no loss at all in classic. Roll seed % 100 < chance. New durability = min(cur-1, max); when cur-1 < 1: a non-weapon not yet broken goes to 0x55d660 (broken flag 0x100, stat list removed); otherwise durability is set to 0 (stackables decrement quantity and replenish).

## 4. 0x62f100 is TRADE_CalcItemPrice; replenish - VERIFIED
Repair mode (6 == 3), durability items that are not ammo/throwable: numerator = max - cur (free when max <= cur). With stat 0xfc (replenish) the numerator is max - 1 (constant, not max-1-cur) and it is free once cur >= max - 1. Price = numerator * price / max. Ethereal items are not repairable.

## 5. 0x63ec50 INV_CheckHandItemsCompatible - VERIFIED
Shoots/quiver pair ok; a lone quiver refused; two-handed items need the barbarian one-or-two-handed test; two weapons (both type 0x2d): class 4 always ok, class 6 needs BOTH of type 0x43 (h2h, h2h2 by equivalence), others refused. Weapon with non-weapon ok; two non-weapons refused. Go fix: right-hand sword + left claw now refused for assassins.

## 6. Ethereal and sockets at generation - VERIFIED
- 0x554d90: skipped by request flag 0x2; needs weapon (0x2d) or armor (0x32), durability applicable, quality not 1 and not 5, not a quest item; rand(100) < 5 (request flag 0x4 or 0x400000 forces, roll still taken); sets flag 0x400000 (0x660a40); max durability = base/2 + 1, cur = max. A +50% ethereal damage/defense is not in this function (unverified, not implemented).
- Order (itemgen.md 1.8): ethereal for qualities 1..9, sockets for 1..3.
- Implemented in d2drop/extras.go (pure) and diablo2item/item_extras.go behind DropOptions.RollExtras (default off; generator seeded from the item seed, so the drop stream and other items are unchanged). Classic-format variants not modelled; Spec rebuild does not persist sockets or halved durability yet.
