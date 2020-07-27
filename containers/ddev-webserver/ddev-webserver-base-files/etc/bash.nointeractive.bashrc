# This file is loaded in non-interactive bash shells through $BASH_ENV
for f in /etc/bashrc/*.bashrc; do
  source $f;
done
