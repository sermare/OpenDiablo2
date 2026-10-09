# OpenDiablo2 on macOS (Apple Silicon) with real Diablo II 1.14b data

This fork runs natively on an M-series Mac. You need your own copy of Diablo II
and Lord of Destruction; no game files are included or distributed here.

## 1. Get the game data

OpenDiablo2 reads the original MPQ archives. On a Mac the easiest way to obtain
them is Blizzard's official downloader, run once under Wine:

1. Download Blizzard's Diablo II (and Lord of Destruction) installers from your
   Battle.net account. They are Windows `.exe` downloaders.
2. Install Wine (`brew install --cask wine-stable`) and run each downloader in a
   fresh prefix: `WINEPREFIX=~/.wine-d2 wine Downloader_Diablo2_enUS.exe`, then
   run the generated `Installer.exe` (and the Lord of Destruction one) the same
   way, installing to the same folder.
3. You should now have `d2data.mpq`, `d2exp.mpq`, `d2char.mpq`, `d2music.mpq`,
   `d2sfx.mpq`, `d2speech.mpq`, `d2video.mpq`, `d2xmusic.mpq`, `d2xtalk.mpq`,
   `d2xvideo.mpq` and `patch_d2.mpq` in
   `~/.wine-d2/drive_c/Program Files (x86)/Diablo II/`.

## 2. Build

```sh
brew install go
go build -o od2 .
```

## 3. Configure

Run `./od2` once; it writes
`~/Library/Application Support/OpenDiablo2/config.json`. Set `MpqPath` to the
folder with the archives and list the MPQs in `MpqLoadOrder` (patch first):

```json
"MpqLoadOrder": ["patch_d2.mpq", "d2exp.mpq", "d2xmusic.mpq", "d2xtalk.mpq",
  "d2xvideo.mpq", "d2data.mpq", "d2char.mpq", "d2music.mpq", "d2sfx.mpq",
  "d2video.mpq", "d2speech.mpq"],
"MpqPath": "/Users/you/.wine-d2/drive_c/Program Files (x86)/Diablo II/"
```

## Testing without the UI

Some environment variables make changes testable without clicking through the
menus (launch from a GUI session, e.g. `open script.command`; running the binary
from a non-GUI shell fails with a Cocoa display error):

| Variable | Effect |
|---|---|
| `OD2_AUTOGAME=<file>` | Start that character directly. A `.d2s` file is imported first. |
| `OD2_D2S_DIR=<dir>` | Import real `.d2s` characters from a folder into the character list. |
| `OD2_AUTOTALK=Warriv,Akara` | After a few seconds, resolve and log each NPC's greeting voice line. |
| `OD2_AUTOTEST_MUTE=1` | Do not play sound during the autotest. |
| `OD2_AUTOEXIT=1` | Quit when the autotest finishes. |

## Real character saves

`d2common/d2fileformats/d2s` reads the header, quests, waypoints, NPC flags,
stats and skills of a Diablo II 1.14b `.d2s`; `HeroStateFactory.ImportD2S`
turns one into an OpenDiablo2 hero. Items are not imported yet.
