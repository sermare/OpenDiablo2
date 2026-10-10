package d2missile

import "testing"

// TestModelledFuncsAgree keeps ModelledHitFuncs in step with Sim.hitFunc.
func TestModelledFuncsAgree(t *testing.T) {
	for id := 1; id < 64; id++ {
		s := &Sim{}
		m := &Missile{Spec: &Spec{SrvHitFunc: id}}

		ok := func() (ok bool) {
			defer func() {
				if recover() != nil {
					ok = true // ran into the world: the function exists
				}
			}()

			_, ok = s.hitFunc(m, nil)

			return ok
		}()

		if ok != contains(ModelledHitFuncs, id) {
			t.Errorf("hit func %d: hitFunc ok=%v, listed=%v", id, ok, contains(ModelledHitFuncs, id))
		}
	}
}
