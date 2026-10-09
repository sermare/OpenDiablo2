//go:build !darwin
// +build !darwin

package main

// runBundleSupervisor is only meaningful inside the macOS .app.
func runBundleSupervisor() (handled bool, code int) { return false, 0 }
