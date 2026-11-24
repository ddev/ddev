# Builds the experimental DDEV desktop app. `make desktop` in the repository
# root builds ddev and this app. The app holds no DDEV code and no ddev
# binary: it runs `ddev serve --desktop` from the installed ddev.

GOTMP ?= ../.gotmp
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
OUT := $(GOTMP)/bin/$(GOOS)_$(GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty)
# macOS wants a bundle version of up to three numbers, such as 1.25.3 from
# v1.25.3-12-gabcdef; DDEVVersion in Info.plist keeps the full version.
BUNDLE_VERSION := $(or $(shell echo '$(VERSION)' | sed -nE 's/^v?([0-9]+\.[0-9]+(\.[0-9]+)?).*/\1/p'),0.0.0)
LDFLAGS := -X 'main.version=$(VERSION)' -s -w

# DDEV.app is one universal bundle for both Mac architectures.
MAC_OUT := $(GOTMP)/bin/darwin_universal
APP := $(MAC_OUT)/DDEV.app
MAC_ARCHS ?= arm64 amd64

# Wails needs CGO and the platform's webview headers on macOS and Linux, so
# those build natively; on macOS, Go passes -arch for the other architecture.
# Windows needs no CGO and cross-builds anywhere.
ifeq ($(GOOS),windows)
CGO := 0
EXE := .exe
LDFLAGS += -H windowsgui
else
CGO := 1
EXE :=
# Wails' Linux C code calls GTK 4's deprecated X11 functions.
export CGO_CFLAGS ?= -O2 -g -Wno-deprecated-declarations
endif

SIGNING_TOOLS_RAW_URL ?= https://raw.githubusercontent.com/ddev/signing_tools/main
DDEV_MACOS_CERT_NAME ?= Developer ID Application: DDEV Foundation (9HQ298V2BW)
DDEV_MACOS_APPLE_ID ?= notarizer@ddev.com
DDEV_MACOS_TEAM_ID ?= 9HQ298V2BW

.PHONY: all binary test app zip sign notarize

ifeq ($(GOOS),darwin)
all: app zip
else
all: binary
endif

# On Linux and Windows the app lands next to ddev in $(OUT), as packages
# install it, and runs that ddev.
binary:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO) go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/ddev-desktop$(EXE) .

# Set DDEV_BINARY to a ddev with `ddev serve` to run the tests that start it.
test:
	CGO_ENABLED=$(CGO) go test -v -count=1 ./...

# A bundle gives macOS an identity for notifications and the Dock. It holds
# only ddev-desktop, which runs the ddev on the login shell's PATH. It is
# signed ad hoc here; `sign` replaces that with the Developer ID.
app:
	for a in $(MAC_ARCHS); do $(MAKE) GOOS=darwin GOARCH=$$a binary || exit 1; done
	rm -rf "$(APP)" "$(MAC_OUT)/DDEV.iconset"
	mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources" "$(MAC_OUT)/DDEV.iconset"
	lipo -create $(foreach a,$(MAC_ARCHS),$(GOTMP)/bin/darwin_$(a)/ddev-desktop) -output "$(APP)/Contents/MacOS/ddev-desktop"
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s icons/appicon.png --out "$(MAC_OUT)/DDEV.iconset/icon_$${s}x$${s}.png" >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) icons/appicon.png --out "$(MAC_OUT)/DDEV.iconset/icon_$${s}x$${s}@2x.png" >/dev/null; \
	done
	iconutil -c icns "$(MAC_OUT)/DDEV.iconset" -o "$(APP)/Contents/Resources/DDEV.icns"
	rm -rf "$(MAC_OUT)/DDEV.iconset"
	sed -e 's/@VERSION@/$(VERSION)/' -e 's/@BUNDLE_VERSION@/$(BUNDLE_VERSION)/' Info.plist > "$(APP)/Contents/Info.plist"
	codesign --force --sign - "$(APP)"
	@echo "Built $(APP)"

# A zip made by ditto keeps the bundle's permissions and signature, which
# an upload-artifact zip of the bare bundle would lose. It zips the bundle as
# it is, so that a signed one stays signed.
zip:
	rm -f "$(MAC_OUT)/DDEV.app.zip"
	ditto -c -k --keepParent "$(APP)" "$(MAC_OUT)/DDEV.app.zip"

# Developer ID signing, with the variables main-build.yml sets for ddev.
sign:
	@if [ -z "$(DDEV_MACOS_SIGNING_PASSWORD)" ] || [ -z "$(DDEV_MACOS_CERT_FILE)" ]; then \
		echo "DDEV_MACOS_SIGNING_PASSWORD and DDEV_MACOS_CERT_FILE are required"; exit 1; fi
	curl -sSL --retry 5 --retry-delay 5 --retry-connrefused --retry-all-errors -f $(SIGNING_TOOLS_RAW_URL)/macos_sign.sh | \
		bash -s - --signing-password="$(DDEV_MACOS_SIGNING_PASSWORD)" --cert-file="$(DDEV_MACOS_CERT_FILE)" --cert-name="$(DDEV_MACOS_CERT_NAME)" --target-binary="$(APP)"
	codesign --verify --deep --strict "$(APP)"

notarize:
	@if [ -z "$(DDEV_MACOS_APP_PASSWORD)" ]; then echo "DDEV_MACOS_APP_PASSWORD is required"; exit 1; fi
	curl -sSL --retry 5 --retry-delay 5 --retry-connrefused --retry-all-errors -f $(SIGNING_TOOLS_RAW_URL)/macos_notarize.sh | \
		bash -s - --app-specific-password="$(DDEV_MACOS_APP_PASSWORD)" --apple-id=$(DDEV_MACOS_APPLE_ID) --team-id=$(DDEV_MACOS_TEAM_ID) --primary-bundle-id=com.ddev.desktop --target-binary="$(APP)"
	xcrun stapler staple "$(APP)"
	$(MAKE) zip
