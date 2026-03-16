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

.PHONY: build lint test install install-desktop install-icons icons uninstall demotape

build:
	go build -o $(BIN_NAME) .

lint:
	go vet ./...
	go fmt ./...
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.11.2
	golangci-lint run

test:
	go test -race ./...

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
