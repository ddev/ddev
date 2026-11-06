# Builds the experimental DDEV desktop app. Run `make desktop` from the
# repository root, which builds ddev first and bundles it with the app.

GOTMP ?= ../.gotmp
OS := $(shell go env GOOS)
ARCH := $(shell go env GOARCH)
OUT := $(GOTMP)/bin/$(OS)_$(ARCH)
VERSION ?= $(shell git describe --tags --always --dirty)
DDEV_BIN ?= $(OUT)
LDFLAGS := -X 'github.com/ddev/ddev/pkg/versionconstants.DdevVersion=$(VERSION)' -X 'main.version=$(VERSION)' -s -w

.PHONY: all binary app

ifeq ($(OS),darwin)
all: app
else
all: binary
endif

# Wails needs CGO on macOS and Linux, and WebKitGTK headers on Linux.
binary:
	CGO_ENABLED=1 go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/ddev-desktop .

# A bundle gives macOS an identity for notifications and the Dock. The ddev
# next to the app's executable is the one it runs.
app: binary
	rm -rf "$(OUT)/DDEV.app" "$(OUT)/DDEV.iconset"
	mkdir -p "$(OUT)/DDEV.app/Contents/MacOS" "$(OUT)/DDEV.app/Contents/Resources" "$(OUT)/DDEV.iconset"
	cp $(OUT)/ddev-desktop $(DDEV_BIN)/ddev $(DDEV_BIN)/ddev-hostname "$(OUT)/DDEV.app/Contents/MacOS/"
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s icons/appicon.png --out "$(OUT)/DDEV.iconset/icon_$${s}x$${s}.png" >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) icons/appicon.png --out "$(OUT)/DDEV.iconset/icon_$${s}x$${s}@2x.png" >/dev/null; \
	done
	iconutil -c icns "$(OUT)/DDEV.iconset" -o "$(OUT)/DDEV.app/Contents/Resources/DDEV.icns"
	rm -rf "$(OUT)/DDEV.iconset"
	sed 's/@VERSION@/$(VERSION)/' Info.plist > "$(OUT)/DDEV.app/Contents/Info.plist"
	codesign --force --deep --sign - "$(OUT)/DDEV.app"
	@echo "Built $(OUT)/DDEV.app"
