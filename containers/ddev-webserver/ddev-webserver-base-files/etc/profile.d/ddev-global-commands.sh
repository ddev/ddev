# Add DDEV global commands (addons) to PATH for login shells
if [ -d /mnt/ddev-global-cache/global-commands/web ]; then
  case ":$PATH:" in
    *":/mnt/ddev-global-cache/global-commands/web:"*) ;;
    *) PATH="$PATH:/mnt/ddev-global-cache/global-commands/web" ;;
  esac
  export PATH
fi
