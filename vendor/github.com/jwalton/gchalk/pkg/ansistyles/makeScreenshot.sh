# To use:
#
#   npm install -g svg-term-cli
#   brew install asciinema
#
svg-term --command='go test -run TestScreenshot | head -n 3' --out=screenshot.svg --padding=3 --width=55 --height=3 --at=1000 --no-cursor
