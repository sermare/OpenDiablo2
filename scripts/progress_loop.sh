#!/bin/zsh
export PATH=/opt/homebrew/bin:/usr/local/bin:/usr/bin:/bin:/Library/Frameworks/Python.framework/Versions/3.12/bin
while true; do
  "$HOME/git/od2-progress/scripts/progress_tick.sh" >> /tmp/od2-progress-tick.log 2>&1
  sleep 600
done
