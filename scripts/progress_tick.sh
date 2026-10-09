#!/bin/zsh
# every 10 min: take the latest docs/progress.json from master, append a snapshot, push branch progress-history
cd "$HOME/git/od2-progress" || exit 1
git fetch -q origin || exit 1
git checkout -q progress-history
git checkout -q origin/master -- docs/progress.json scripts/progress_history.py
python3 scripts/progress_history.py tick
git add -A docs scripts
git commit -qm "progress-history: snapshot $(date -u +%FT%TZ)" && git push -q origin progress-history
