#!/bin/zsh
# Flake counter: scripts/flake_loop.sh <scenario-glob> <runs> [parallel=1]
# Runs scripts/verify.sh for one scenario <runs> times (<parallel> at a time) under the usual machine-wide slot cap and
# counts first-attempt failures (the "RETRY:" line) and final failures. Output: /tmp/od2-flakeloop.<scenario>.<n>.txt
# Defaults the data env (nokkasorc sample, d2-tables) and needs OD2_VERIFY_CONFIG_DIR (private config folder).
set -u
cd "${0:A:h}/.."
sc=$1; runs=$2; par=${3:-1}
export D2S_SAMPLE_BODY=${D2S_SAMPLE_BODY:-$HOME/git/nokka-d2s-ref/examples/nokkasorc}
export D2S_SAMPLE_BODY_JSON=${D2S_SAMPLE_BODY_JSON:-$HOME/git/nokka-d2s-ref/examples/nokkasorc.json}
export D2_TABLES=${D2_TABLES:-$HOME/git/d2-tables}
export OD2_VERIFY_CONFIG_DIR=${OD2_VERIFY_CONFIG_DIR:-$PWD/cfgw36}
export SKIP_UNIT=1 SKIP_BUDGET=1 OD2_MAX_GAMES=${OD2_MAX_GAMES:-3} OD2_VERIFY_ONLY="$sc"
export OD2_VERIFY_FLAKE_DIR=${OD2_VERIFY_FLAKE_DIR:-/tmp/od2-flake-w36}
tag=${OD2_FLAKE_TAG:-run}
out=/tmp/od2-flakeloop.${tag}.${sc//[^a-z0-9-]/_}
setopt nullglob
rm -f $out.*.txt
export SKIP_HYGIENE=1
# OD2_FLAKE_BURNER=1 adds a CPU burner (one busy loop per 3 cores) for the length of the loop
burn=()
if [ -n "${OD2_FLAKE_BURNER:-}" ]; then for i in 1 2 3; do (while :; do :; done) & burn+=($!); done; fi
trap 'for b in $burn; do kill $b 2>/dev/null; done' EXIT
# one build for the whole loop (a build on a loaded machine takes minutes and once failed)
bin=$(mktemp /tmp/od2-flakebin.XXXXXX)
go build -o $bin . 2>&1 | grep -v "ld: warning"
[ ${pipestatus[1]} -eq 0 ] || { echo "BUILD FAILED"; rm -f $bin; exit 1; }
export OD2_VERIFY_BIN=$bin
trap 'rm -f $bin; for b in $burn; do kill $b 2>/dev/null; done' EXIT
one() { scripts/verify.sh > $out.$1.txt 2>&1; echo "run $1 exit=$?"; }
n=1
while [ $n -le $runs ]; do
  jobs_p=(); for ((k=0;k<par && n<=runs;k++,n++)); do one $n & jobs_p+=($!); done
  wait $jobs_p # only these jobs: a bare wait also waits for the CPU burners (infinite loops)
done
retry=$(grep -l "^RETRY:" $out.*.txt 2>/dev/null | wc -l | tr -d ' ')
bad=$(grep -L "ALL CHECKS PASSED" $out.*.txt 2>/dev/null | wc -l | tr -d ' ')
echo "FLAKELOOP $sc runs=$runs first_attempt_failed=$retry final_failed=$bad"
