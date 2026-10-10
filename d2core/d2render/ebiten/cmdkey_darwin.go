package ebiten

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

static int cmdHeld(void) {
	return (CGEventSourceFlagsState(kCGEventSourceStateCombinedSessionState) & kCGEventFlagMaskCommand) ? 1 : 0;
}
*/
import "C"

// commandHeld reports whether a Command key is down. ebiten 2.0 has no key for it
// (GLFW's super key), so the system's modifier state is read directly; this is
// safe from any thread.
func commandHeld() bool { return C.cmdHeld() != 0 }
