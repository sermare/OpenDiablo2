#!/bin/zsh
# Tests scripts/verify_classes.sh without a game: a fake verify.sh stands in (OD2_CLASS_MATRIX_VERIFY) and the
# runner's gate, result parsing (PASS, PASS* after a retry, FAIL), death count, first anomaly, resume and the
# generated docs/class-matrix.md are checked. Nothing here starts the game or touches the repository's docs.
#   scripts/test_verify_classes.sh
set -u
cd "${0:A:h}/.." || exit 1

t=$(mktemp -d /tmp/od2-classmatrix-test.XXXXXX)
trap 'rm -rf $t' EXIT
fail=0
check() { if eval "$2"; then echo "ok   $1"; else echo "FAIL $1"; fail=1; fi; }

# fake verify.sh: the outcome depends on OD2_HERO and OD2_VERIFY_ONLY, the scratch folder and log mimic the real one
cat > $t/fake-verify.sh <<'EOF'
#!/bin/zsh
tmp=$(mktemp -d /tmp/od2-verify.XXXXXX)
echo "$tmp" > "$OD2_VERIFY_TMPFILE"
log=$tmp/$OD2_VERIFY_ONLY.log.txt
echo "game started" > $log
case "$OD2_HERO:$OD2_VERIFY_ONLY" in
  necro:9e-act2-playthrough)
    echo "DEATH hero=NokkaNecro at Rocky Waste" >> $log
    echo "DEATH hero=NokkaNecro at Dry Hills" >> $log
    echo "FAIL: the level 94 hero died on the way"
    echo "SOME CHECKS FAILED"; echo "SCENARIOS RUN: 1"; exit 1 ;;
  amazon:9b-act1-playthrough)
    echo "RETRY: Act 1 playthrough failed once; running it again"
    echo "ALL CHECKS PASSED"; echo "SCENARIOS RUN: 1"; exit 0 ;;
  druid:9g-act3-playthrough)
    echo "[WARNING] KILL left skill dropped" >> $log
    echo "FAIL: warnings/errors in the Act 3 log"
    echo "SOME CHECKS FAILED"; echo "SCENARIOS RUN: 1"; exit 1 ;;
  assassin:*) echo "VERIFY ABORTED: D2_TABLES is unset"; exit 1 ;;
esac
echo "ALL CHECKS PASSED"; echo "SCENARIOS RUN: 1"; exit 0
EOF
chmod +x $t/fake-verify.sh

export OD2_CLASS_MATRIX_DIR=$t/work OD2_CLASS_MATRIX_DOC=$t/matrix.md OD2_CLASS_MATRIX_VERIFY=$t/fake-verify.sh
export OD2_CLASS_MATRIX_FORCE=1   # the fake starts no game, so another verify_parallel.sh on the machine does not matter
export D2_TABLES=$t D2S_SAMPLE_BODY=$t/x D2S_SAMPLE_BODY_JSON=$t/x
touch $t/x
unset OD2_VERIFY_NIGHTLY

# the gate: without OD2_VERIFY_NIGHTLY nothing runs, exit 0
out=$(./scripts/verify_classes.sh --classes necro 2>&1); rc=$?
check "no OD2_VERIFY_NIGHTLY: skipped with exit 0" '[ $rc -eq 0 ] && echo "$out" | grep -q SKIPPED && [ ! -s $t/work/results.tsv ]'

# a dry run lists the commands, needs no gate and starts nothing
out=$(./scripts/verify_classes.sh --dry-run --classes necro,sorc --scenarios 9b-act1-playthrough 2>&1)
check "dry run lists one command per pair" '[ $(echo "$out" | grep -c "^OD2_HERO=") -eq 2 ]'

# a run: necro dies in Act 2, amazon passes 9b on the retry, druid fails on a warning, sorc passes everything
export OD2_VERIFY_NIGHTLY=1
./scripts/verify_classes.sh --classes necro,amazon,druid,sorc --scenarios 9b-act1-playthrough,9e-act2-playthrough,9g-act3-playthrough > $t/run.out 2>&1
rc=$?
[ $rc -ne 1 ] && cat $t/run.out
r=$t/work/results.tsv
check "the run fails (a FAIL row exists)" '[ $rc -eq 1 ]'
check "12 rows, one per pair" '[ $(wc -l < $r | tr -d " ") -eq 12 ]'
check "necro 9e is FAIL with 2 deaths and the FAIL line as anomaly" 'grep -q "^necro.9e-act2-playthrough.FAIL.[0-9]*.2.FAIL: the level 94 hero died on the way" $r'
check "necro 9b is PASS with 0 deaths and no anomaly" 'grep -q "^necro.9b-act1-playthrough.PASS.[0-9]*.0.$" $r'
check "amazon 9b is PASS* (passed on the retry)" 'grep -q "^amazon.9b-act1-playthrough.PASS\*" $r'
check "druid 9g is FAIL and names the warning" 'grep -q "^druid.9g-act3-playthrough.FAIL.[0-9]*.0.FAIL: warnings/errors" $r'
check "sorc passes all three" '[ $(grep -c "^sorc.*.PASS" $r) -eq 3 ]'
left=0; for f in $t/work/tmpdir.*; do [ -d "$(cat $f)" ] && left=1; done
check "the scratch folders of the runs were removed" '[ $left -eq 0 ]'
check "the game log of a failed run was kept" '[ -s $t/work/necro.9e-act2-playthrough.log.txt ]'

# the document
d=$t/matrix.md
check "doc: title and the table header" 'grep -q "^# Class playthrough matrix" $d && grep -q "^| class | 9b Act 1 (fresh) | 9e Act 2 | 9g Act 3 |" $d'
check "doc: sorc row shows three passes" 'grep -q "^| sorc | PASS .* PASS .* PASS .* | .* | 0 | 3/3 |" $d'
check "doc: necro row shows the failure and 2 deaths" 'grep "^| necro |" $d | grep -q "FAIL .* d2 .* | 2 | 2/3 |"'
check "doc: the anomaly table lists the necro failure" 'grep -q "^| necro | 9e-act2-playthrough | FAIL .* the level 94 hero died" $d'

# --resume re-runs only what did not pass
: > $t/calls
cat > $t/count-verify.sh <<'EOF'
#!/bin/zsh
echo "$OD2_HERO $OD2_VERIFY_ONLY" >> "$CALLS"
exec "$FAKE"
EOF
chmod +x $t/count-verify.sh
CALLS=$t/calls FAKE=$t/fake-verify.sh OD2_CLASS_MATRIX_VERIFY=$t/count-verify.sh \
  ./scripts/verify_classes.sh --resume --classes necro,amazon,druid,sorc --scenarios 9b-act1-playthrough,9e-act2-playthrough,9g-act3-playthrough > $t/resume.out 2>&1
check "resume runs only the two failed pairs" '[ $(wc -l < $t/calls | tr -d " ") -eq 2 ] && grep -q "necro 9e-act2-playthrough" $t/calls && grep -q "druid 9g-act3-playthrough" $t/calls'
check "resume keeps one row per pair" '[ $(wc -l < $r | tr -d " ") -eq 12 ]'

# an aborted verify (assassin) is a FAIL with the abort line
./scripts/verify_classes.sh --classes assassin --scenarios 9b-act1-playthrough > $t/abort.out 2>&1
check "an aborted verify is a FAIL naming the abort" 'grep -q "^assassin.9b-act1-playthrough.FAIL.[0-9]*.0.VERIFY ABORTED" $r'

# --render alone rewrites the doc without any run
rm -f $d
./scripts/verify_classes.sh --render > /dev/null 2>&1
check "--render writes the doc from the table" '[ -s $d ] && grep -q "Class playthrough matrix" $d'

[ $fail -eq 0 ] && echo "ALL verify_classes TESTS PASSED" || { echo "verify_classes TESTS FAILED"; exit 1; }
