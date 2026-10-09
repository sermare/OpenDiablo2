// Package d2key parses the Diablo II 1.14b default.key command table.
//
// Layout (verified against the real 1146 byte file): a 6 byte header made of
// three little-endian u16 (magic 'WS' = 0x5357, version 0x25, total file size
// 0x47a) followed by 57 records of 20 bytes.
//
// Record (little-endian):
//
//	+0  u32 command id
//	+4  u16 primary Windows virtual-key code (0xffff = unbound)
//	+6  u16 flag A (meaning unverified; 1 for most primary keys)
//	+8  u16 padding
//	+10 u16 command id repeated
//	+12 u16 padding
//	+14 u16 secondary virtual-key code (0xffff = unbound)
//	+16 u32 flag B (meaning unverified)
//
// Command ids are not stored in id order (id 21 is the 46th record);
// lookups are by id. Codes 0x100-0x104 are not Windows VK codes: the real
// game uses them for mouse input (0x103/0x104 are the wheel; 0x100-0x102 are
// unverified).
package d2key
