#!/usr/bin/env bash
# Shared helpers for perf/ metric scripts. Source, don't execute:
#   source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"

# now_ms prints the current epoch time in milliseconds. Falls back gracefully
# on platforms where GNU date's %N isn't available (e.g. stock BSD date on macOS).
now_ms() {
  local candidate
  candidate="$(date +%s%3N 2>/dev/null || true)"
  if [[ "$candidate" =~ ^[0-9]+$ ]]; then
    echo "$candidate"
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c 'import time; print(int(time.time()*1000))'
  elif command -v perl >/dev/null 2>&1; then
    perl -MTime::HiRes=time -e 'printf "%d\n", time()*1000'
  else
    echo $(( $(date +%s) * 1000 ))
  fi
}

# median reads newline-separated integers on stdin and prints their median.
median() {
  sort -n | awk '{a[NR]=$1} END{n=NR; if(n==0){print 0; exit} if(n%2==1){print a[(n+1)/2]}else{print int((a[n/2]+a[n/2+1])/2)}}'
}

# emit_metric <name> <value_ms> converts value_ms to seconds and prints one JSON
# result line to stdout. value_ms may be the literal "null" when a metric
# doesn't apply (e.g. Mutagen disabled).
emit_metric() {
  local name="$1" value="$2" value_s="null"
  if [[ "$value" != "null" ]]; then
    value_s=$(awk -v ms="$value" 'BEGIN { printf "%.1f", ms / 1000 }')
  fi
  jq -n --arg metric "$name" --argjson value_s "$value_s" '{metric: $metric, value_s: $value_s}'
}
