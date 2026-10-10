//go:build !darwin

package ebiten

func commandKeyDown() bool { return false }
