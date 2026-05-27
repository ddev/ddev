# This file is loaded in non-interactive bash shells through $BASH_ENV

for f in /etc/bashrc/*.bashrc; do
  source $f;
done
unset f

# Source user-specific bashrc additions.
for f in "$HOME/.bashrc.d/"*; do
  [ -f "$f" ] && source "$f"
done
unset f
