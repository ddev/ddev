# This file is loaded in non-interactive bash shells through $BASH_ENV
for f in /etc/bashrc/*.bashrc; do
  source $f;
done
unset f

for i in $(\ls $HOME/.bashrc.d/* 2>/dev/null); do
    source $i;
done
unset i
