# Git hooks

Hook scripts in this directory can be placed in .git/hooks to get git to help with our workflow. These are for developer use only, and have no impact by just being here in .githooks.

You should also be able to link them, for example (if you don't mind if they change upstream, and don't introduce any changes of your own)
```
cd .git/hooks
# For all the static checks:
ln -s ../../.githooks/pre-push.allchecks pre-push
# or for just gofmt quick check
ln -s ../../.githooks/pre-push.gofmt pre-push
```