package d2player

// ExeDefaultKey is one row of the default key table built into Game.exe 1.14b
// (CMDTBL_InitKeyBindings 0x466460 copies 0x474 bytes from 0x711b60 to the live
// table at 0x79f018 when no usable key file exists; 114 entries of 10 bytes, the
// primary and the secondary entry of each of the 57 commands, VERIFIED). Codes
// are Windows virtual keys; 0x100..0x104 are the game's own mouse codes (read as
// Mouse 3, Mouse 4, Mouse 5, wheel up, wheel down: VERIFIED from the jump table
// at 0x465c38 of CMDTBL_GetCommandNameString 0x465760, strings 0xeb6 0xeb7 0xeb8
// 0xeb9 0xeba) and 0xffff is unbound. Commands 46..53 are the extra skill hotkeys 9..16 of the
// expansion list; 57 (0x39) marks a separator row in the key configuration table.
type ExeDefaultKey struct {
	Command            uint32
	Primary, Secondary uint16
}

// exeUnbound is the code of an empty slot.
const exeUnbound = 0xffff

// ExeDefaultKeys lists the 57 commands in the exe's table order.
var ExeDefaultKeys = [57]ExeDefaultKey{
	{0, 'A', 'C'}, {1, 'B', 'I'}, {2, 'P', exeUnbound}, {3, 'M', exeUnbound}, {4, 'Q', exeUnbound},
	{5, 0x0d, exeUnbound}, {6, 'H', exeUnbound}, {7, 0x09, 0x100},
	{8, 0x78, exeUnbound}, {9, 0x79, exeUnbound}, {10, 0x7a, exeUnbound}, {11, 0x7b, exeUnbound},
	{12, 'T', exeUnbound}, {13, 'S', exeUnbound},
	{14, 0x70, exeUnbound}, {15, 0x71, exeUnbound}, {16, 0x72, exeUnbound}, {17, 0x73, exeUnbound},
	{18, 0x74, exeUnbound}, {19, 0x75, exeUnbound}, {20, 0x76, exeUnbound},
	{22, 0xc0, exeUnbound}, {23, '1', exeUnbound}, {24, '2', exeUnbound}, {25, '3', exeUnbound}, {26, '4', exeUnbound},
	{27, 0x60, exeUnbound}, {28, 0x61, exeUnbound}, {29, 0x62, exeUnbound}, {30, 0x63, exeUnbound},
	{31, 0x64, exeUnbound}, {32, 0x65, exeUnbound}, {33, 0x66, exeUnbound},
	{34, 0x11, exeUnbound}, {35, 'R', 0x102}, {36, 0x10, exeUnbound}, {37, 0x12, 0x101},
	{38, 0x20, exeUnbound}, {39, 0x103, exeUnbound}, {40, 0x104, exeUnbound},
	{41, 'N', exeUnbound}, {42, 0x2c, exeUnbound}, {43, 'Z', exeUnbound}, {44, 'W', exeUnbound}, {45, 'V', exeUnbound},
	{21, 0x77, exeUnbound},
	{46, exeUnbound, exeUnbound}, {47, exeUnbound, exeUnbound}, {48, exeUnbound, exeUnbound},
	{49, exeUnbound, exeUnbound}, {50, exeUnbound, exeUnbound}, {51, exeUnbound, exeUnbound},
	{52, exeUnbound, exeUnbound}, {53, exeUnbound, exeUnbound},
	{54, 'O', exeUnbound}, {55, 0x67, exeUnbound}, {56, 0x1b, exeUnbound},
}

// ExeKeyConfigGroups is the layout of the key configuration panel (table at
// 0x723528; each row is {command, string id, ...}; rows with command 0x39 are the
// gaps between groups). VERIFIED from the table; the string texts were not read.
// Groups of the plain game: panels 0 1 2 3 4 6 / skills 12 13 14..21 39 40 / belt
// 22..26 / run and chat 5 34 35 36 37 43 / automap 7 8 9 10 11 / say 27..33 55 /
// screen 42 38 41. The expansion list inserts 54 (hireling, string 0x58c7) after 2
// and ends the skill group with 46..53 (hotkeys 9..16).
var ExeKeyConfigGroups = [][]uint32{
	{0, 1, 2, 3, 4, 6},
	{12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 39, 40},
	{22, 23, 24, 25, 26},
	{5, 34, 35, 36, 37, 43},
	{7, 8, 9, 10, 11},
	{27, 28, 29, 30, 31, 32, 33, 55},
	{42, 38, 41},
}

// ExeAutomapKeyRows are the rows of the automap group of the key configuration table (0x723528), VERIFIED by
// reading the table: {command, string id}. The string ids are CfgAutoMap (0xf5a), CfgAutoMapCenter (0xf5b),
// CfgAutoMapFade (0xf8f), CfgAutoMapParty (0xf91) and CfgAutoMapNames (0xf90) in string.tbl, so commands 8..11
// (default F9..F12) are Center, Fade, Party, Names, in that order. (w44-unv-ui listed the ids sorted; the table
// order is Party before Names.)
var ExeAutomapKeyRows = [5]struct {
	Command  uint32
	StringID uint16
	Key      string
}{
	{7, 0xf5a, "CfgAutoMap"}, {8, 0xf5b, "CfgAutoMapCenter"}, {9, 0xf8f, "CfgAutoMapFade"},
	{10, 0xf91, "CfgAutoMapParty"}, {11, 0xf90, "CfgAutoMapNames"},
}
