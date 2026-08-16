INSTALL_DIR ?= $(HOME)/.local
BIN_DIR ?= $(INSTALL_DIR)/bin
APP_DIR ?= $(INSTALL_DIR)/share/applications
ICON_DIR ?= $(INSTALL_DIR)/share/icons/hicolor
CACHE_DIR ?= $(HOME)/.cache/tmus
CONFIG_DIR ?= $(HOME)/.config/tmus

BIN_NAME ?= tmus
DESKTOP_FILE ?= packaging/tmus.desktop
ICON_BASE ?= packaging/icons/hicolor
ICON_SOURCE ?= packaging/icons/source/tmus.png

# Linux supplies the baseline runtime graph. The explicit packages add the
# components that are linked only on macOS or Windows, yielding the reviewed
# conservative union for all release targets.
NOTICE_PACKAGES := . github.com/ebitengine/purego github.com/inconshreveable/mousetrap
NOTICE_REPORT := go tool go-licenses report $(NOTICE_PACKAGES) --ignore github.com/bpicode/tmus
NOTICE_CHECK := go tool go-licenses check $(NOTICE_PACKAGES) --ignore github.com/bpicode/tmus --ignore github.com/llehouerou/go-m4a

.PHONY: build lint test notices notices-check install install-desktop install-icons icons uninstall demotape

build:
	go build -o $(BIN_NAME) .

lint:
	go vet ./...
	go fmt ./...
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run

test:
	go test -race ./...

notices:
	$(NOTICE_CHECK)
	$(NOTICE_REPORT) --template=third_party_notices.tpl > THIRD_PARTY_NOTICES.md
	$(NOTICE_REPORT) --template=packaging/debian/copyright.tpl > packaging/debian/copyright

notices-check:
	$(NOTICE_CHECK)
	$(NOTICE_REPORT) --template=third_party_notices.tpl | diff -u THIRD_PARTY_NOTICES.md -
	$(NOTICE_REPORT) --template=packaging/debian/copyright.tpl | diff -u packaging/debian/copyright -

install: install-desktop
	mkdir -p $(BIN_DIR)
	go build -ldflags="-s -w" -o $(BIN_DIR)/$(BIN_NAME) .

install-desktop: install-icons
	mkdir -p $(APP_DIR)
	install -m 0644 $(DESKTOP_FILE) $(APP_DIR)/tmus.desktop

install-icons: icons
	mkdir -p $(ICON_DIR)/48x48/apps
	mkdir -p $(ICON_DIR)/256x256/apps
	mkdir -p $(ICON_DIR)/512x512/apps
	install -m 0644 $(ICON_BASE)/48x48/apps/tmus.png $(ICON_DIR)/48x48/apps/tmus.png
	install -m 0644 $(ICON_BASE)/256x256/apps/tmus.png $(ICON_DIR)/256x256/apps/tmus.png
	install -m 0644 $(ICON_BASE)/512x512/apps/tmus.png $(ICON_DIR)/512x512/apps/tmus.png

icons:
	go run ./tools/genicon -png $(ICON_SOURCE) -size 48 -out $(ICON_BASE)/48x48/apps/tmus.png
	go run ./tools/genicon -png $(ICON_SOURCE) -size 256 -out $(ICON_BASE)/256x256/apps/tmus.png
	go run ./tools/genicon -png $(ICON_SOURCE) -size 512 -out $(ICON_BASE)/512x512/apps/tmus.png

uninstall:
	rm -f $(BIN_DIR)/$(BIN_NAME)
	rm -f $(APP_DIR)/tmus.desktop
	rm -f $(ICON_DIR)/48x48/apps/tmus.png
	rm -f $(ICON_DIR)/256x256/apps/tmus.png
	rm -f $(ICON_DIR)/512x512/apps/tmus.png
	rm -rf $(CACHE_DIR)
	rm -rf $(CONFIG_DIR)

demotape:
	podman run --rm --device /dev/snd --entrypoint /bin/bash -v $(PWD):/vhs ghcr.io/charmbracelet/vhs -c "cd /vhs && apt update && apt install libasound2-dev && vhs demo.tape"
