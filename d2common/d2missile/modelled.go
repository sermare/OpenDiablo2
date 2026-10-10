package d2missile

// ModelledHitFuncs lists the pSrvHitFunc ids (0x739a68 table) Sim.hitFunc
// implements; the skills audit (d2core/d2records/skills_audit_test.go) reads
// it. TestModelledFuncsAgree keeps it in step with the switch in hitfunc.go.
var ModelledHitFuncs = []int{1, 2, 3, 4, 7, 8, 9, 10, 12, 13, 14, 20, 26, 27, 29, 36, 47, 48, 51, 56}

// ModelledDoFuncs lists the pSrvDoFunc ids (per-frame movement function) the
// Sim steps: 1 plain flight, 2 and 6 trail missiles, 7 homing (Guided Arrow),
// 27 Tornado, and (funcs.go, all read from the exe) 3 and 5 cell stamp (5 also
// the fire animation frame), 10 and 25 random sub missiles (Blizzard,
// Eruption), 14 periodic helper (Grim Ward), 23 and 24 sub missile per new
// cell (Firestorm), 28 Volcano scatter.
var ModelledDoFuncs = []int{1, 2, 3, 5, 6, 7, 10, 14, 15, 23, 24, 25, 27, 28}

// ApproxDoFuncs are movement functions the Sim flies as plain flight (1)
// although the exe does more. None any more: 3 and 5 only stamp the missile
// bit 0x40 into the collision cell (and 5 animates), which is modelled through
// CellMarker.
var ApproxDoFuncs = []int{}

// DoFuncApprox reports whether a movement function is flown as plain flight.
func DoFuncApprox(id int) bool { return contains(ApproxDoFuncs, id) }

// HitFuncModelled reports whether the sim implements a missile hit function
// (0 is "none" and always fine).
func HitFuncModelled(id int) bool { return id == 0 || contains(ModelledHitFuncs, id) }

// DoFuncModelled reports whether the sim implements a missile movement
// function (0 is "none" and always fine).
func DoFuncModelled(id int) bool { return id == 0 || contains(ModelledDoFuncs, id) }

func contains(l []int, v int) bool {
	for _, x := range l {
		if x == v {
			return true
		}
	}

	return false
}
