package ebiten

/*
#cgo darwin LDFLAGS: -framework CoreGraphics
#include <CoreGraphics/CoreGraphics.h>

static int od2CommandDown(void) {
	return (CGEventSourceFlagsState(kCGEventSourceStateCombinedSessionState) & kCGEventFlagMaskCommand) != 0;
}
*/
import "C"

// commandKeyDown reports whether a Command key is held (ebiten v2.0.2 has no
// key constant for it).
func commandKeyDown() bool { return C.od2CommandDown() != 0 }
