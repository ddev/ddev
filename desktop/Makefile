# Builds the experimental DDEV desktop app. Run `make desktop` from the
# repository root, which builds ddev first and bundles it with the app.

GOTMP ?= ../.gotmp
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
OUT := $(GOTMP)/bin/$(GOOS)_$(GOARCH)
VERSION ?= $(shell git describe --tags --always --dirty)
DDEV_BIN ?= $(OUT)
APP := $(OUT)/DDEV.app
LDFLAGS := -X 'github.com/ddev/ddev/pkg/versionconstants.DdevVersion=$(VERSION)' -X 'main.version=$(VERSION)' -s -w

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
endif

SIGNING_TOOLS_RAW_URL ?= https://raw.githubusercontent.com/ddev/signing_tools/main
DDEV_MACOS_CERT_NAME ?= Developer ID Application: DDEV Foundation (9HQ298V2BW)
DDEV_MACOS_APPLE_ID ?= notarizer@ddev.com
DDEV_MACOS_TEAM_ID ?= 9HQ298V2BW

.PHONY: all binary app zip sign notarize

ifeq ($(GOOS),darwin)
all: app zip
else
all: binary
endif

binary:
	GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO) go build -trimpath -ldflags "$(LDFLAGS)" -o $(OUT)/ddev-desktop$(EXE) .

# A bundle gives macOS an identity for notifications and the Dock. The ddev
# next to the app's executable is the one it runs. It is signed ad hoc here;
# `sign` replaces that with the Developer ID.
app: binary
	rm -rf "$(APP)" "$(OUT)/DDEV.iconset"
	mkdir -p "$(APP)/Contents/MacOS" "$(APP)/Contents/Resources" "$(OUT)/DDEV.iconset"
	cp $(OUT)/ddev-desktop $(DDEV_BIN)/ddev $(DDEV_BIN)/ddev-hostname "$(APP)/Contents/MacOS/"
	for s in 16 32 128 256 512; do \
		sips -z $$s $$s icons/appicon.png --out "$(OUT)/DDEV.iconset/icon_$${s}x$${s}.png" >/dev/null; \
		sips -z $$((s*2)) $$((s*2)) icons/appicon.png --out "$(OUT)/DDEV.iconset/icon_$${s}x$${s}@2x.png" >/dev/null; \
	done
	iconutil -c icns "$(OUT)/DDEV.iconset" -o "$(APP)/Contents/Resources/DDEV.icns"
	rm -rf "$(OUT)/DDEV.iconset"
	sed 's/@VERSION@/$(VERSION)/' Info.plist > "$(APP)/Contents/Info.plist"
	codesign --force --deep --sign - "$(APP)"
	@echo "Built $(APP)"

# A zip made by ditto keeps the bundle's permissions and signature, which
# an upload-artifact zip of the bare bundle would lose. It zips the bundle as
# it is, so that a signed one stays signed.
zip:
	rm -f "$(OUT)/DDEV.app.zip"
	ditto -c -k --keepParent "$(APP)" "$(OUT)/DDEV.app.zip"

# Developer ID signing, with the variables main-build.yml sets for ddev.
# The executables inside the bundle are signed before the bundle itself.
sign:
	@if [ -z "$(DDEV_MACOS_SIGNING_PASSWORD)" ] || [ -z "$(DDEV_MACOS_CERT_FILE)" ]; then \
		echo "DDEV_MACOS_SIGNING_PASSWORD and DDEV_MACOS_CERT_FILE are required"; exit 1; fi
	set -e; for f in "$(APP)/Contents/MacOS/ddev" "$(APP)/Contents/MacOS/ddev-hostname" "$(APP)"; do \
		curl -sSL --retry 5 --retry-delay 5 --retry-connrefused --retry-all-errors -f $(SIGNING_TOOLS_RAW_URL)/macos_sign.sh | \
			bash -s - --signing-password="$(DDEV_MACOS_SIGNING_PASSWORD)" --cert-file="$(DDEV_MACOS_CERT_FILE)" --cert-name="$(DDEV_MACOS_CERT_NAME)" --target-binary="$$f"; \
	done
	codesign --verify --deep --strict "$(APP)"

notarize:
	@if [ -z "$(DDEV_MACOS_APP_PASSWORD)" ]; then echo "DDEV_MACOS_APP_PASSWORD is required"; exit 1; fi
	curl -sSL --retry 5 --retry-delay 5 --retry-connrefused --retry-all-errors -f $(SIGNING_TOOLS_RAW_URL)/macos_notarize.sh | \
		bash -s - --app-specific-password="$(DDEV_MACOS_APP_PASSWORD)" --apple-id=$(DDEV_MACOS_APPLE_ID) --team-id=$(DDEV_MACOS_TEAM_ID) --primary-bundle-id=com.ddev.desktop --target-binary="$(APP)"
	xcrun stapler staple "$(APP)"
	rm -f "$(OUT)/DDEV.app.zip"
	ditto -c -k --keepParent "$(APP)" "$(OUT)/DDEV.app.zip"
