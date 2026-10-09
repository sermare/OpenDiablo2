package d2config

import (
	"os"
	"path/filepath"
)

const (
	od2ConfigDirName  = "OpenDiablo2"
	od2ConfigFileName = "config.json"
)

// ConfigDirEnv names an environment variable that overrides the directory that
// holds config.json and the Saves folder. It exists for tests; normal use keeps
// the per-user default (~/Library/Application Support/OpenDiablo2 on macOS).
const ConfigDirEnv = "OD2_CONFIG_DIR"

// ConfigDir returns the directory holding config.json and Saves, or "" when the
// user config directory is unknown.
func ConfigDir() string {
	if dir := os.Getenv(ConfigDirEnv); dir != "" {
		return dir
	}

	if configDir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(configDir, od2ConfigDirName)
	}

	return ""
}

// DefaultConfigPath returns the absolute path for the default config file location
func DefaultConfigPath() string {
	if dir := ConfigDir(); dir != "" {
		return filepath.Join(dir, od2ConfigFileName)
	}

	return LocalConfigPath()
}

// LocalConfigPath returns the absolute path to the directory of the OpenDiablo2 executable
func LocalConfigPath() string {
	return filepath.Join(filepath.Dir(os.Args[0]), od2ConfigFileName)
}
