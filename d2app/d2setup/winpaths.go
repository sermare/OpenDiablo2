package d2setup

import "strings"

// Pure Windows path logic. Everything here builds backslash paths by hand (not filepath.Join), so it behaves the same on every
// host and the unit tests run on a Mac. The Windows-only callers (registry, environment) are in winpaths_windows.go and
// winpaths_other.go. UNVERIFIED on a real Windows machine: the registry value names below come from memory of the classic
// Diablo II installer, not from a Windows box.

// registryInstallKey is one place the classic installer is believed to record where the game lives.
type registryInstallKey struct {
	Hive  string // "HKLM" or "HKCU"
	Path  string
	Value string
}

// registryInstallKeys lists the keys to try, most specific first. The 32-bit view (WOW6432Node) comes first because the
// classic game is a 32-bit program on 64-bit Windows.
//
//nolint:gochecknoglobals // constant table
var registryInstallKeys = []registryInstallKey{
	{"HKLM", `SOFTWARE\WOW6432Node\Blizzard Entertainment\Diablo II`, "InstallPath"},
	{"HKLM", `SOFTWARE\Blizzard Entertainment\Diablo II`, "InstallPath"},
	{"HKCU", `Software\Blizzard Entertainment\Diablo II`, "InstallPath"},
	// Steam's own folder; the game would be in steamapps\common below it.
	{"HKCU", `Software\Valve\Steam`, "SteamPath"},
}

// registrySavePathKeys are values that may hold the folder of the real .d2s characters.
//
//nolint:gochecknoglobals // constant table
var registrySavePathKeys = []registryInstallKey{
	{"HKCU", `Software\Blizzard Entertainment\Diablo II`, "NewSavePath"},
	{"HKLM", `SOFTWARE\WOW6432Node\Blizzard Entertainment\Diablo II`, "NewSavePath"},
}

// gameSubdirs are the folders a Diablo II install is usually in, below a Program Files style directory.
//
//nolint:gochecknoglobals // constant table
var gameSubdirs = []string{
	"Diablo II", "Diablo II Classic", `Steam\steamapps\common\Diablo II`, `Battle.net\Diablo II`,
	`Blizzard Entertainment\Diablo II`,
}

// winJoin joins path elements with backslashes, dropping empty elements and doubled separators. A first element such as
// "C:\" keeps its meaning ("C:\Games").
func winJoin(elem ...string) string {
	var parts []string

	for i, e := range elem {
		e = strings.ReplaceAll(e, "/", `\`)
		if i == 0 {
			e = strings.TrimRight(e, `\`)
		} else {
			e = strings.Trim(e, `\`)
		}

		if e != "" {
			parts = append(parts, e)
		}
	}

	return strings.Join(parts, `\`)
}

// NormalizeInstallPath cleans a registry or config value into a folder: strips quotes, spaces and NULs, turns slashes into
// backslashes, removes a trailing separator and, when the value names the game executable, returns its folder.
func NormalizeInstallPath(raw string) string {
	p := strings.Trim(strings.TrimSpace(strings.Trim(raw, "\x00")), `"`)
	p = strings.ReplaceAll(strings.TrimSpace(p), "/", `\`)

	if i := strings.LastIndex(p, `\`); i >= 0 && strings.HasSuffix(strings.ToLower(p), ".exe") {
		p = p[:i]
	}

	if len(p) > 3 {
		p = strings.TrimRight(p, `\`)
	}

	return p
}

// windowsGameDirsFor lists the usual install folders. drives are like "C:"; programFiles are the directories from
// %ProgramFiles%, %ProgramFiles(x86)% and %ProgramW6432% (empty ones are skipped). Duplicates (ignoring case) are dropped.
func windowsGameDirsFor(drives, programFiles []string) []string {
	var out []string

	add := func(dir string) {
		for _, o := range out {
			if strings.EqualFold(o, dir) {
				return
			}
		}

		out = append(out, dir)
	}

	for _, pf := range programFiles {
		if pf == "" {
			continue
		}

		for _, sub := range gameSubdirs {
			add(winJoin(pf, sub))
		}
	}

	for _, d := range drives {
		for _, pf := range []string{"Program Files (x86)", "Program Files"} {
			for _, sub := range gameSubdirs {
				add(winJoin(d+`\`, pf, sub))
			}
		}

		add(winJoin(d+`\`, "Games", "Diablo II"))
		add(winJoin(d+`\`, "Diablo II"))
		add(winJoin(d+`\`, "SteamLibrary", "steamapps", "common", "Diablo II"))
	}

	return out
}

// windowsInstallsFromRegistry turns registry values into candidate game folders. A Steam root becomes the folder of the
// game inside it.
func windowsInstallsFromRegistry(values map[registryInstallKey]string) []string {
	var out []string

	for _, k := range registryInstallKeys {
		v := NormalizeInstallPath(values[k])
		if v == "" {
			continue
		}

		if k.Value == "SteamPath" {
			out = append(out, winJoin(v, "steamapps", "common", "Diablo II"))

			continue
		}

		out = append(out, v)
	}

	return out
}

// windowsSaveGlobs lists the folders (glob patterns) with real characters, for the user profile directory and the optional
// OneDrive folder (OneDrive can redirect Saved Games).
func windowsSaveGlobs(userProfile, oneDrive string) []string {
	var out []string

	for _, root := range []string{userProfile, oneDrive} {
		if root == "" {
			continue
		}

		out = append(out, winJoin(root, "Saved Games", "Diablo II*"), winJoin(root, "Saved Games", "Diablo II*", "Save"))
	}

	return out
}

// isResurrectedDir reports whether a save folder belongs to Diablo II: Resurrected, whose .d2s files use another format.
func isResurrectedDir(dir string) bool {
	return strings.Contains(strings.ToLower(dir), "resurrected")
}
