//go:build windows

package d2setup

import (
	"os"

	"golang.org/x/sys/windows/registry"
)

func registryValues(keys []registryInstallKey) map[registryInstallKey]string {
	out := map[registryInstallKey]string{}

	for _, k := range keys {
		root := registry.LOCAL_MACHINE
		if k.Hive == "HKCU" {
			root = registry.CURRENT_USER
		}

		key, err := registry.OpenKey(root, k.Path, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		if s, _, err := key.GetStringValue(k.Value); err == nil {
			out[k] = s
		}

		key.Close()
	}

	return out
}

func windowsEnvDirs() (drives, programFiles []string) {
	drives = []string{"C:", "D:", "E:", "F:"}

	for _, e := range []string{"ProgramFiles(x86)", "ProgramFiles", "ProgramW6432"} {
		programFiles = append(programFiles, os.Getenv(e))
	}

	return drives, programFiles
}

func windowsSaveHomes() (userProfile, oneDrive string) {
	return os.Getenv("USERPROFILE"), os.Getenv("OneDrive")
}
