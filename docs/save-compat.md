# Save-file compatibility

What the `d2s` package reads and writes, what it refuses, and how to move an older save forward.

## Supported

| Item | Status |
| --- | --- |
| `.d2s` versions 0x5C-0x60 (1.10-1.14d) | parse -> write is byte-identical (tested on real 0x60 saves; the other versions by flipping the version field of a real save, so only 0x60 is verified against a real game file) |
| Classic and expansion, softcore, hardcore, dead, ladder | round trip tested on every combination |
| All difficulty/act states of the header | round trip tested (3 difficulties x 5 acts) |
| Mercenary header + `jf` item list, corpse (`JM` + 12 byte header), iron golem (`kf`) | parsed, written, preserved |
| Unknown bytes (header filler, quest/waypoint padding, trailing data) | kept verbatim; `d2s.Unsupported(c)` lists what is carried blindly |
| Bad size or checksum | `d2s.Repair` / `d2s.ParseRepair` fix them and report what changed; `Write` always recomputes both |
| Damaged or truncated files | never panic: `Parse` returns a regular error (a panic inside the parser is converted to `ErrCorrupt`). Fuzz tests truncate at every length and corrupt random bytes |

Tests: `go test ./d2common/d2fileformats/d2s/` with `D2_TABLES` and `D2S_SAMPLE_BODY` set. `TestRealSavesRoundTrip` also scans
`~/git/d2s-test`, the nokka examples, the Wine saves folders and `D2S_SAVE_DIRS` (a path list) for real saves, hashes the re-written bytes
against the original and skips if none exist. Saves are only read; no save is ever committed.

## Not supported

* Versions below 0x5C use the legacy layouts: 0x47 (1.00-1.05), 0x57 (1.06-1.07), 0x59 (1.08-1.09). `ParseHeader` rejects them with
  `ErrBadVersion` whose message names the release. Versions above 0x60 (Resurrected) are rejected too.
* Resurrected `.d2s`/`.d2i` files: different item encoding.
* Writing stash files: the stash import is read-only.

## Migrating an older save

Old saves cannot be edited in place by this code. Supported route:

1. Open the character once in Diablo II 1.14 (any 1.14 build): the game converts 1.09 and older files to the 1.14 layout
   (0x60) on first load and save. 1.09-1.10 files that lack the expansion data load as classic characters.
2. Copy the converted `.d2s` into the save folder of this port (or point the import at it).
3. If the file was edited by hand and now fails with `checksum mismatch`, run it through `d2s.Repair` (size field and checksum only).

Direct conversion of the legacy bit layouts (item bit widths and stat tables differ in 0x47/0x57/0x59) is not implemented.

## Shared stash import (read-only)

`d2s.ParseStash` reads PlugY stash files: shared stash `.sss` (softcore) / `.hss` (hardcore), magic `SSS\0`, and personal stash `.d2x`
(`.stash`), magic `CSTM`. A page is `ST`, optional flags and name, then an ordinary `JM` item list. **The header layout is reconstructed
from memory of PlugY and unverified: no real stash existed on the development machine.** The parser trusts only its tags and the item
lists and reports the offset of anything else. Set `D2S_STASH_FILES` (path list) to run it on real files.

A Resurrected `.d2i` is recognised and refused with `ErrStashFormat`.

In the game, `OD2_STASH_IMPORT=/path/to/file.sss` adds the file's items to the stash of every hero imported from a `.d2s`
(`d2hero.MergeStash`): items keep their grid cell when it is free, otherwise they take the first free place; ears, unknown base items
and items that do not fit are skipped and counted in the `stash import:` log line. The stash file itself is never written.
