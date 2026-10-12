//go:build !windows

package d2setup

func registryValues([]registryInstallKey) map[registryInstallKey]string { return nil }

func windowsEnvDirs() (drives, programFiles []string) { return nil, nil }

func windowsSaveHomes() (userProfile, oneDrive string) { return "", "" }
