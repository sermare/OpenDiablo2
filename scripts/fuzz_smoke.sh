#!/usr/bin/env bash
# Short fuzz run over every Go native fuzz target (testing.F) in the pure packages.
#
#   scripts/fuzz_smoke.sh                 # every target, FUZZTIME (default 10s) each
#   FUZZTIME=30s scripts/fuzz_smoke.sh    # longer
#   scripts/fuzz_smoke.sh ./d2common/d2fileformats/d2mpq   # only the targets of these packages
#
# Seed corpora are generated synthetically inside the tests (f.Add); no game files are read or
# committed. A failing input is written by the go tool to <pkg>/testdata/fuzz/<Target>/ ; copy it
# into the repo as a regression seed once the bug is fixed (the files are tiny, synthetic mutations).
# Exit status is non-zero when any target fails or when a package fails to build.
set -u

cd "$(dirname "$0")/.."

FUZZTIME="${FUZZTIME:-10s}"
# Fuzz workers share the machine with other jobs: keep parallelism modest.
PARALLEL="${FUZZ_PARALLEL:-2}"

if [ "$#" -gt 0 ]; then
  pkgs="$*"
else
  pkgs="$(go list ./d2common/... ./d2networking/... ./d2game/d2autoscript/... ./d2script/... 2>/dev/null)"
fi

fail=0
ran=0
failed_list=""

for pkg in $pkgs; do
  targets="$(go test -list '^Fuzz' "$pkg" 2>/dev/null | grep '^Fuzz' || true)"
  for t in $targets; do
    ran=$((ran + 1))
    echo "=== fuzz $pkg $t ($FUZZTIME)"
    out="$(mktemp)"
    if go test "$pkg" -run '^$' -fuzz "^${t}\$" -fuzztime "$FUZZTIME" -parallel "$PARALLEL" >"$out" 2>&1; then
      grep -E '^(ok|PASS|fuzz: elapsed: [0-9.]+s.*)$' "$out" | tail -n 1
    else
      grep -v 'ignoring duplicate libraries' "$out" | tail -n 40
      echo "FAIL: $pkg $t (failing input, if any, is in the package's testdata/fuzz/$t)"
      fail=$((fail + 1))
      failed_list="$failed_list $pkg:$t"
    fi
    rm -f "$out"
  done
done

echo "fuzz_smoke: $ran targets run, $fail failed${failed_list:+ ($failed_list)}"
[ "$fail" -eq 0 ]
