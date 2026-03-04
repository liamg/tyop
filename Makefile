APP     = tyop.app
BINARY  = $(APP)/Contents/MacOS/tyop
DEST    = /Applications/$(APP)
VERSION ?= dev

.PHONY: build install uninstall clean

build:
	go build -ldflags="-X main.version=$(VERSION)" -o $(BINARY) .

install: build
	rm -rf $(DEST)
	cp -r $(APP) $(DEST)
	open $(DEST)
	@echo ""
	@echo "Installed to $(DEST)"
	@echo "If prompted, grant Accessibility access in:"
	@echo "  System Settings → Privacy & Security → Accessibility"

uninstall:
	launchctl unload ~/Library/LaunchAgents/com.liamg.tyop.plist 2>/dev/null || true
	rm -f ~/Library/LaunchAgents/com.liamg.tyop.plist
	rm -rf $(DEST)
	@echo "Uninstalled."

clean:
	rm -f $(BINARY)
