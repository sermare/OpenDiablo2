package d2missile

// ModelledHitFuncs lists the pSrvHitFunc ids (0x739a68 table) Sim.hitFunc
// implements; the skills audit (d2core/d2records/skills_audit_test.go) reads
// it. TestModelledFuncsAgree keeps it in step with the switch in hitfunc.go.
var ModelledHitFuncs = []int{1, 2, 3, 4, 7, 10, 12, 13, 14, 20, 29, 36}

// ModelledDoFuncs lists the pSrvDoFunc ids (per-frame movement function) the
// Sim steps: 1 plain flight, 2 and 6 trail missiles, 7 homing (Guided Arrow),
// 27 Tornado.
var ModelledDoFuncs = []int{1, 2, 6, 7, 15, 27}

// ApproxDoFuncs are movement functions the Sim flies as plain flight (1):
// 3 only stamps the "missile here" collision bit 0x40 per cell (0x5abfb0), 5
// is an animation wobble (0x5ac050, UNVERIFIED role). Neither changes damage
// or path in the sim.
var ApproxDoFuncs = []int{3, 5}

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
