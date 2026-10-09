//go:build !darwin
// +build !darwin

package ebiten

// commandHeld: there is no Command key outside macOS.
func commandHeld() bool { return false }
