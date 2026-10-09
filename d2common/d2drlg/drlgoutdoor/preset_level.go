package drlgoutdoor

import (
	"errors"
	"fmt"

	"github.com/OpenDiablo2/OpenDiablo2/d2common/d2math/d2rand"
)

// PresetLevel is a generated DrlgType 2 level (DRLG_GeneratePresetLevel,
// 0x66ad50): the file index and one room per 8x8 chunk of the level rectangle.
type PresetLevel struct {
	ID    int
	Def   int // LvlPrest Def of the level
	File  int // DS1 file index (Roll(Files); the town of Act 1 is forced by the world)
	Rect  Rect
	Rooms []*Room
	Seed  *d2rand.Seed // level seed after the rooms
	// GateSeed is the level seed when the DS1 objects were filtered (the draws
	// of drlgpop.Filter start from it).
	GateSeed d2rand.Seed
}

// Counts of the DS1 object RNG gates (DRLG_FilterPresetObjects, 0x66a230).
// monstatsCount is the number of monstats rows the exe had loaded in the
// oracle harness; the expansion id base (FUN_006571b0) is UNVERIFIED (0).
const (
	monstatsCount = 734
	expansionBase = 0
)

// FilterPresetObjects is the object gate pass of a preset DS1: it advances the
// level seed once for every object with a gated id and returns the number of
// steps. In Acts 4/5 no DS1 contains a gated id (checked for all 349 reachable
// files, drlg-act45-outdoor.md section 5), so it is a no-op there; the hook
// keeps the seed stream right for other files. Which objects get dropped is
// not needed by the seed and not tracked.
func FilterPresetObjects(seed *d2rand.Seed, d *Pattern) int {
	n := 0

	for _, o := range d.Objects {
		switch o.Type {
		case 1:
			if o.ID < monstatsCount {
				switch o.ID {
				case 0xcc, 0xcd, 0x173, 0x174:
					seed.Step()
					n++
				}

				continue
			}

			if k := o.ID - monstatsCount - expansionBase; k >= 0 && (k == 0x21 || k == 0x22 || k == 0x23) {
				seed.Step()
				n++
			}
		case 2:
			switch objectID(d, o.ID) {
			case 0xc4, 0x105, 0x245:
				seed.Step()
				n++
			}
		}
	}

	return n
}

// objectTable is the exe table at 0x744af8 (750 entries): DS1 type 2 object id
// below 0x96 of act a -> in-memory object id, indexed a*0x96 + id.
var objectTable = [750]uint16{
	12, 37, 39, 35, 36, 5, 17, 18, 19, 20, 21, 22, 30, 70, 70,
	69, 69, 29, 31, 33, 34, 37, 61, 65, 66, 8, 26, 28, 82, 2,
	81, 84, 83, 78, 61, 103, 108, 119, 580, 130, 159, 163, 169, 160, 161,
	162, 104, 105, 106, 107, 179, 180, 119, 157, 247, 248, 155, 174, 175, 139,
	140, 141, 144, 6, 240, 241, 242, 54, 55, 56, 57, 58, 171, 178, 239,
	245, 250, 111, 138, 132, 164, 165, 77, 85, 86, 262, 263, 264, 265, 50,
	51, 79, 53, 1, 3, 7, 46, 38, 256, 257, 258, 129, 267, 268, 269,
	581, 351, 352, 353, 374, 385, 397, 321, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	74, 37, 192, 304, 305, 306, 101, 102, 78, 103, 156, 580, 132, 129, 357,
	153, 121, 122, 229, 230, 196, 267, 261, 149, 269, 4, 9, 52, 94, 95,
	142, 143, 5, 6, 87, 88, 146, 146, 147, 148, 240, 241, 242, 243, 176,
	177, 198, 246, 29, 160, 161, 162, 273, 283, 85, 86, 109, 116, 134, 135,
	136, 150, 151, 172, 173, 279, 280, 281, 282, 166, 167, 113, 137, 89, 104,
	105, 106, 107, 154, 171, 178, 270, 271, 272, 266, 274, 244, 284, 288, 298,
	289, 296, 297, 287, 286, 285, 290, 291, 292, 293, 294, 295, 133, 303, 299,
	300, 301, 302, 581, 354, 582, 314, 315, 316, 317, 323, 322, 110, 112, 114,
	355, 356, 357, 351, 352, 353, 152, 374, 387, 389, 390, 391, 388, 397, 402,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	117, 237, 580, 130, 102, 37, 160, 161, 162, 104, 105, 106, 107, 194, 195,
	193, 207, 211, 210, 234, 214, 215, 213, 228, 216, 227, 217, 235, 218, 219,
	220, 221, 223, 224, 267, 269, 581, 170, 325, 184, 190, 191, 197, 199, 200,
	201, 202, 206, 278, 120, 130, 326, 158, 271, 272, 327, 328, 329, 330, 331,
	332, 333, 334, 335, 336, 5, 6, 176, 240, 241, 181, 183, 246, 185, 186,
	187, 188, 203, 204, 205, 208, 209, 169, 323, 324, 196, 212, 225, 244, 351,
	352, 353, 360, 361, 362, 365, 251, 252, 208, 283, 367, 366, 368, 341, 342,
	343, 344, 374, 370, 378, 379, 386, 397, 405, 407, 406, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	238, 580, 267, 269, 581, 573, 573, 573, 345, 346, 347, 348, 349, 350, 351,
	352, 353, 358, 359, 363, 259, 373, 372, 374, 236, 249, 226, 231, 232, 93,
	97, 123, 124, 96, 225, 233, 222, 125, 126, 127, 128, 375, 376, 254, 253,
	342, 255, 392, 393, 394, 395, 396, 398, 397, 399, 401, 400, 380, 383, 384,
	296, 297, 403, 102, 408, 409, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	452, 453, 338, 337, 267, 374, 482, 39, 35, 36, 33, 34, 38, 102, 411,
	438, 412, 435, 436, 440, 441, 441, 442, 429, 420, 431, 430, 413, 432, 433,
	418, 419, 424, 425, 416, 414, 415, 427, 428, 421, 422, 423, 426, 451, 267,
	443, 444, 445, 446, 447, 448, 450, 451, 459, 460, 461, 462, 482, 473, 455,
	456, 457, 458, 463, 464, 465, 466, 467, 468, 469, 470, 471, 472, 477, 479,
	480, 481, 483, 484, 485, 486, 487, 454, 437, 508, 488, 493, 493, 495, 497,
	499, 503, 509, 512, 489, 490, 514, 515, 493, 498, 513, 494, 496, 511, 500,
	501, 502, 504, 505, 506, 507, 510, 160, 161, 162, 269, 523, 434, 496, 496,
	496, 527, 528, 538, 539, 542, 543, 546, 547, 548, 549, 550, 551, 552, 555,
	553, 554, 557, 541, 544, 559, 560, 564, 567, 568, 536, 537, 563, 570, 397,
}

// waypointObjects are the in-memory object ids of the 16 waypoints.
var waypointObjects = map[int]bool{119: true, 145: true, 156: true, 157: true, 237: true, 238: true, 288: true, 323: true,
	324: true, 398: true, 402: true, 429: true, 494: true, 496: true, 511: true, 539: true}

// objectID maps a DS1 type 2 object id to the in-memory object id
// (DRLG_ParseDS1Data, version >= 6): ids below 0x96 go through objectTable.
func objectID(d *Pattern, id int) int {
	if id < 0x96 {
		act := d.Act
		if act > 4 {
			act = 4
		}

		return int(objectTable[act*0x96+id])
	}

	return id - 0x96
}

// waypointChunk reports whether the DS1 has a waypoint object in the 8x8 chunk.
func waypointChunk(d *Pattern, cx, cy int) bool {
	for _, o := range d.Objects {
		if o.Type != 2 {
			continue
		}

		id := objectID(d, o.ID)
		if id >= 0 && id < 0x23d && waypointObjects[id] && (o.X/5)>>3 == cx && (o.Y/5)>>3 == cy {
			return true
		}
	}

	return false
}

// GeneratePreset runs DRLG_GeneratePresetLevel for a DrlgType 2 level.
// Params carry the level rectangle and the registered vis/warp arrays (see
// ParamsFromLayout45). A non-negative fileOverride replaces the rolled file
// index (the Act 1 world forces the town's).
func GeneratePreset(env *Env, p Params, fileOverride int) (res *PresetLevel, err error) {
	defer func() {
		if r := recover(); r != nil {
			res, err = nil, fmt.Errorf("drlgoutdoor: preset level %d: %v", p.ID, r)
		}
	}()

	rec, ok := env.Tables.PrestByLevel(p.ID)
	if !ok {
		return nil, fmt.Errorf("drlgoutdoor: no LvlPrest row for level %d", p.ID)
	}

	l := &Level{Params: p, env: env}
	l.Seed = d2rand.New(p.BaseSeed + uint32(p.ID))

	// DRLG_AllocPresetMap: one level-seed step whose value is the file index
	// (DRLG_InitPresetLevel did the same on a seed that is re-initialised).
	file := 0
	if rec.Files > 0 {
		file = int(l.Seed.Roll(int32(rec.Files)))
	}

	if fileOverride >= 0 {
		file = fileOverride
	}

	rect := p.Rect
	if rec.SizeX != 0 && rec.SizeY != 0 {
		rect.W, rect.H = rec.SizeX, rec.SizeY
	}

	var flags0 uint32
	if rec.Outdoors != 0 {
		flags0 = 0x80000
	}

	for k := 0; k < 8; k++ {
		if p.Vis[k] != 0 && p.Warp[k] == -1 {
			flags0 |= 0x10 << uint(k)
		}
	}

	var ds *Pattern

	gateSeed := *l.Seed

	if rec.Scan != 0 || rec.Pops != 0 {
		if file < 0 || file >= len(rec.File) {
			return nil, errors.New("drlgoutdoor: preset file index out of range")
		}

		if f := rec.File[file]; f != "" {
			if ds, err = env.Pattern(NormalizePrestFile(f)); err != nil {
				return nil, err
			}

			FilterPresetObjects(l.Seed, ds)
		}
	}

	out := &PresetLevel{ID: p.ID, Def: rec.Def, File: file, Rect: rect, Seed: l.Seed, GateSeed: gateSeed}

	remY := rect.H

	for cy := rect.Y; cy < rect.Y+rect.H; cy += 8 {
		remX := rect.W

		for cx := rect.X; cx < rect.X+rect.W; cx += 8 {
			r := l.allocRoom(2)
			r.X, r.Y, r.W, r.H = cx, cy, min(8, remX), min(8, remY)
			r.PrestDef, r.File = rec.Def, file
			r.Flags = flags0

			if ds != nil {
				chx, chy := (cx-rect.X)/8, (cy-rect.Y)/8
				r.Flags |= presetChunkBits(ds, chx, chy)

				if waypointChunk(ds, chx, chy) {
					r.Flags |= 0x30000
				}
			}

			if rec.Populate == 0 {
				r.Flags |= 0x800000
			}

			r.R50 = uint32(rec.Dt1Mask)
			out.Rooms = append(out.Rooms, r)
			remX -= 8
		}

		remY -= 8
	}

	return out, nil
}
