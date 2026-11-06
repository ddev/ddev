#!/bin/bash
# Read-only: report each socktainer worktree's branch vs rfay remote and upstream.
cd ~/workspace/socktainer || exit 1
git fetch -q rfay
git fetch -q upstream
git worktree list --porcelain | awk '/^worktree /{print $2}' | while read -r d; do
  b=$(git -C "$d" rev-parse --abbrev-ref HEAD)
  if git rev-parse --verify -q "rfay/$b" >/dev/null; then
    r=$(git rev-list --left-right --count "rfay/$b...$b" | awk '{print "remote-only=" $1 " local-only=" $2}')
  else
    r="NOT-ON-REMOTE"
  fi
  dirty=$(git -C "$d" status --porcelain | wc -l | tr -d ' ')
  up=$(git rev-list --count "upstream/main..$b")
  echo "$d | $b | $r | dirty=$dirty | commits-not-in-upstream=$up"
done
