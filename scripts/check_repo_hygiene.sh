#!/usr/bin/env bash
# Fails if the git tree contains game data or Blizzard decompiled source.
#   - forbidden extensions: .mpq .d2s .dc6 .dt1 .ds1 .dcc .cof .wav .tbl
#   - files larger than 2 MB (except under d2common/d2drlg/testdata and the allowlist
#     in scripts/repo_hygiene_allowlist.txt)
#   - Ghidra-decompiler output patterns (iVar1, undefined4, unaff_*, ...) and
#     Blizzard copyright headers. Plain addresses and names such as FUN_00453190, DAT_00710da8, param_1 in comments
#     are allowed: they are notes, not code.
# Usage: scripts/check_repo_hygiene.sh        (checks tracked + staged files; runs from anywhere)
set -u
cd "$(dirname "$0")/.." || exit 2
allow=scripts/repo_hygiene_allowlist.txt
fail=0
is_allowed() { grep -v '^#' "$allow" 2>/dev/null | grep -qxF -- "$1"; }

files=$( { git ls-files; git diff --cached --name-only --diff-filter=AM; } | sort -u )

echo "== forbidden extensions"
while IFS= read -r f; do
  [ -f "$f" ] || continue
  case "$(printf '%s' "$f" | tr 'A-Z' 'a-z')" in
    *.mpq|*.d2s|*.dc6|*.dt1|*.ds1|*.dcc|*.cof|*.wav|*.tbl)
      if is_allowed "$f"; then echo "allowed: $f"; else echo "FORBIDDEN: $f"; fail=1; fi ;;
  esac
done <<< "$files"

echo "== files over 2 MB"
while IFS= read -r f; do
  [ -f "$f" ] || continue
  size=$(wc -c < "$f" | tr -d ' ')
  if [ "$size" -gt 2097152 ]; then
    case "$f" in d2common/d2drlg/testdata/*) continue ;; esac
    if is_allowed "$f"; then echo "allowed: $f ($size bytes)"; else echo "TOO LARGE: $f ($size bytes)"; fail=1; fi
  fi
done <<< "$files"

echo "== Blizzard decompiled-source patterns"
pat='\b[iupcsl]Var[0-9]+\b|\bundefined[1248]\b|\bunaff_|\bin_stack_|\bswitchD_|\bcode \*\*\)|Copyright[^a-z]*(\(c\)|©)?[^a-z]*[0-9]{4}.*Blizzard'
hits=""
while IFS= read -r f; do
  [ -f "$f" ] || continue
  case "$f" in scripts/check_repo_hygiene.sh) continue ;; esac
  is_allowed "$f" && continue
  h=$(grep -InE "$pat" "$f" 2>/dev/null | grep -v '^Binary' | head -3 | sed "s|^|$f:|")
  [ -n "$h" ] && hits="$hits$h"$'\n'
done <<< "$files"
if [ -n "$hits" ]; then printf '%s' "$hits" | head -20; echo "DECOMPILER-STYLE CODE FOUND"; fail=1; fi

if [ "$fail" -eq 0 ]; then echo "repo hygiene: OK"; else echo "repo hygiene: FAILED"; fi
exit $fail
