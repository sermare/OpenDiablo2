# OpenDiablo2 on Windows (experimental, never run on Windows yet)

Refreshed in wave 36 from the wave 35 tree: `go vet` clean for windows/amd64, PE32+ rebuilt.

Status, stated plainly: the Windows `.exe` is **cross-compiled from a Mac and has never been started on a Windows
machine**. What was checked: `go vet` for `GOOS=windows GOARCH=amd64` is clean, the build succeeds, and `file` reports a
PE32+ x86-64 executable. What was not checked: everything at run time (window, input, audio, the first-run dialogs, saves).
Expect rough edges and please report them with the log text.

You need your **own** copy of Diablo II and Lord of Destruction (1.14b files). No game files are in this repository or in
the zip, and none are distributed.

## Run it

1. Unzip `OpenDiablo2-<version>-windows-amd64.zip` anywhere (for example `C:\Games\OpenDiablo2`). It holds only
   `OpenDiablo2.exe`, this README and the licence.
2. Double-click `OpenDiablo2.exe`. SmartScreen may warn about an unsigned program: More info, Run anyway.
3. On the first run the game looks for your Diablo II folder (`C:\Program Files (x86)\Diablo II`, `C:\Program Files\Diablo II`,
   Steam / Battle.net / `Games` folders on C: to F:). If it finds one it asks to use it; otherwise a folder picker (PowerShell
   dialog) opens. Pick the folder that contains `d2data.mpq`, `d2char.mpq`, `d2exp.mpq`, `patch_d2.mpq` and the other `.mpq`
   files. The files are only read.
4. Your real `.d2s` characters are imported (originals never changed) from `%USERPROFILE%\Saved Games\Diablo II*`, the
   `Save` folder inside the game folder and a few other known places.

## config.json

The configuration is read from, and written to, `%APPDATA%\OpenDiablo2\config.json` (that is
`C:\Users\<you>\AppData\Roaming\OpenDiablo2\config.json`; Go's `os.UserConfigDir`). Saves go in the `Saves` folder next to it.
Set the environment variable `OD2_CONFIG_DIR` to use another folder. If that file does not exist the game falls back to a
`config.json` next to the exe.

Minimal settings (JSON accepts forward slashes, which avoids escaping backslashes):

```json
{
  "MpqPath": "C:/Program Files (x86)/Diablo II",
  "MpqLoadOrder": [
    "patch_d2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq", "d2xvideo.mpq",
    "d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq", "d2video.mpq", "d2speech.mpq"
  ]
}
```

Write backslashes as `\\` if you use them. Set `OD2_NO_SETUP=1` to skip the first-run dialogs.

## Paths and case

- The Windows file system is case-insensitive, so `D2DATA.MPQ` and `d2data.mpq` both open. Names inside the MPQ archives are
  lowercased and use backslashes internally; the loader already converts `/` to `\` and the MPQ hash is case-insensitive,
  so nothing platform specific is needed. The case-insensitive open helper is used on Linux only.
- All game-side paths are built with `filepath.Join`, so they get backslashes on Windows. The macOS-only locations in
  first-run discovery are harmless no-ops on Windows.
- Process CPU time (used only by the `OD2_PERF` self-profiler) uses `GetProcessTimes` on Windows.

## Build it yourself (from a Mac or Linux machine)

```
scripts/make-windows.sh            # dist/windows-amd64/OpenDiablo2.exe (console subsystem, log lines visible)
ZIP=1 scripts/make-windows.sh      # also dist/OpenDiablo2-<version>-windows-amd64.zip
WINGUI=1 scripts/make-windows.sh   # no console window (but then log output is invisible)
```

No C compiler is needed (`CGO_ENABLED=0`): the pinned Ebiten v2.0.2 loads an embedded `glfw.dll` and uses `winmm` for audio.
`zig` (installed at `/opt/homebrew/bin/zig` here) or mingw is only needed if Ebiten is upgraded to a version that requires
cgo; then use `CGO_ENABLED=1 CC="zig cc -target x86_64-windows-gnu"`. Nothing is installed by the script.
To check by hand: `GOOS=windows GOARCH=amd64 go vet ./... && file dist/windows-amd64/OpenDiablo2.exe`.

## Windows on ARM

On the pinned Ebiten v2.0.2 `GOARCH=arm64` does **not** build (no embedded `glfw.dll` for arm64, and the old `golang.org/x/sys`
lacks windows/arm64). Windows 11 on ARM runs the amd64 `.exe` under emulation, which is the supported route. With Ebiten
v2.10.5 (wave 36 branch `feat/w36-ebiten-upgrade`, see the section below) `GOOS=windows GOARCH=arm64` builds and vets clean and
produces a PE32+ Aarch64 file, so a native ARM build becomes possible once that branch is merged; it has not been run either.

## Ebiten upgrade (v2.0.2 to v2.10.5)

Findings from the trial on branch `feat/w36-ebiten-upgrade` (wave 36):

- The source needed **no API changes**: all uses (`ebiten`, `audio`, `audio/wav`, `ebitenutil`, `inpututil`) compile unchanged
  against v2.10.5 (deprecated calls still exist).
- `go.mod` moves to `go 1.25.0` (Ebiten's floor). Two consequences, both handled: the newer `go vet` printf check flagged 8
  non-constant format strings (fixed, one of them was a real bug: log text containing `%`), and Go 1.24+ turns `rand.Seed` into a
  no-op, so `go.mod` carries `godebug randseednop=0` to keep the seeded behaviour.
- Windows needs no cgo with v2.10.5 either (it uses purego/`x/sys`), so `CGO_ENABLED=0` still works; macOS uses purego for Metal and
  oto v3 (CoreAudio), so only the repo's own `cmdkey_darwin.go` still needs cgo there.
- Not checked on this machine: audio output differences (new oto backend), cursor and key repeat changes, gamepad mapping
  (the Gamepad API was reworked after 2.0), and anything on Windows itself. See `docs/WINDOWS.md` history and the wave 36 notes
  for the scenario results.

## Known untested or unsupported on Windows

- Full screen: the game now starts full screen every time (`alwaysFullscreen`, unless an `OD2_*` test variable or
  `OD2_WINDOWED=1` is set). On Windows that is Ebiten's borderless full screen at the monitor size; Alt+Enter toggles it (a
  duplicate toggle that cancelled itself was removed in wave 36), and the `fullscreen` console command saves the choice.
- Saves are written through a temp file and renamed over the target; on Windows the rename is retried for about 200 ms because
  another process holding the file open (antivirus, a second game) makes it fail briefly.
- `OD2_AUTOCAST` test heroes use `os.TempDir()` (`%TEMP%`); the `/tmp/od2-slots` machine cap belongs to the Mac test scripts only.

- First-run dialogs (`d2app/d2setup/ui_native_windows.go`) use PowerShell with `System.Windows.Forms`; the Yes/No box maps
  the two buttons to the "use this / choose another" choices and says so in the text.
- The macOS app supervisor, crash dialog and `.app` logging do not exist here; run from a console to see the log.
- No Wine test was done.
