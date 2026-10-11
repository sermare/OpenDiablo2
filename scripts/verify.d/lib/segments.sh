# Helpers of the real-input scenarios (8g-*): the log is cut into segments at marker lines and each segment's events are counted.
# Source this from a scenario file:  source scripts/verify.d/lib/segments.sh
#
# INTERACT_RE matches the one log line a click on an NPC, door, waypoint, chest, stash or ground item writes when it starts the
# walk to it (game.go OnPlayerInteract, ground_items.go walkTo*, level_change.go useObject): one such line per interaction.
INTERACT_RE='interacting with "|OBJECT walking to use |walking to open |walking to the stash|walking to quest object|walking to pick up '

# seg_events <log.txt> <marker regex> [<extra event regex>]
# Prints, for each segment that starts at a line matching the marker, one line:
#   SEG <n> interact=<count> extra=<count> cast=<count> worldclicks=<action:count,...> hovers=<labels,...>
# (segment 0 is everything before the first marker). interact counts INTERACT_RE lines, extra the optional regex, cast the
# "CAST start" lines, worldclicks the INPUT world-click actions, hovers the HOLD hover labels.
seg_events() {
  tr -d "\r" < $1 | grep -v AUTOSCRIPT | awk -v marker="$2" -v inter="$INTERACT_RE" -v extra="${3:-NOEXTRAPATTERN}" '
    function flush(   k, w) { w=""; for (k in wcn) w = (w==""?"":w ",") k ":" wcn[k]; printf "SEG %d interact=%d extra=%d cast=%d worldclicks=%s hovers=%s\n", n, ni, nx, nc, (w==""?"-":w), (hv==""?"-":hv) }
    BEGIN { n=0 }
    $0 ~ marker { flush(); n++; ni=0; nx=0; nc=0; delete wcn; hv=""; next }
    $0 ~ inter { ni++ }
    $0 ~ extra { nx++ }
    /CAST start/ { nc++ }
    /INPUT world-click/ { a=$0; sub(/.*action=/, "", a); sub(/ .*/, "", a); wcn[a]++ }
    /HOLD hover/ { l=$0; sub(/.*HOLD hover t=[0-9.]+ /, "", l); gsub(/"/, "", l); gsub(/ /, "_", l); if (l != "") hv = (hv==""?l:hv "," l) }
    END { flush() }' | sed '1d'
}

# seg_field <SEG line> <field>   prints the value of a field of a seg_events line
seg_field() { echo "$1" | tr ' ' '\n' | sed -n "s/^$2=//p" | head -1; }

# hold_segments <log.txt>
# For every "HOLD start" prints   HOLD <n> samples=<k> walked=<tiles> longest_still=<seconds>   from the HOLD pos samples
# (about every half second): the distance walked by the hero and the longest time it did not move.
hold_segments() {
  tr -d '\r' < $1 | grep -E "HOLD pos|HOLD start" | sed -E 's/.*HOLD pos t=([0-9.]+) \(([-0-9.]+),([-0-9.]+)\).*/P \1 \2 \3/; s/.*HOLD start.*/S 1/' | awk '
    function report() { printf "HOLD %d samples=%d walked=%.1f longest_still=%.1f\n", seg, n, total, worst }
    /^S/ { if (seg) report(); seg++; n=0; total=0; still=0; worst=0; next }
    /^P/ { if (n>0) { d=sqrt(($3-px)^2+($4-py)^2); total+=d; if (d<0.05) { still+=$2-pt; if (still>worst) worst=still } else still=0 }
           px=$3; py=$4; pt=$2; n++ }
    END { if (seg) report() }'
}
