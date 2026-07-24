PLUGIN := key-bind
PKG := ./cmd/key-bind
DIST := dist
WEB := web
EMBED_INDEX := internal/plugin/web/dist/index.html

UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
	EXT := dylib
else
	EXT := so
endif

.PHONY: all web plugin test clean build-linux-amd64 build-linux-arm64

# Default: build the web UI, then the native plugin binary.
all: web plugin

test:
	go test ./...

# Build the single-file web UI and place it where the Go embed expects it.
web:
	cd $(WEB) && npm install && VITE_HOSTED=1 npm run build
	cp $(WEB)/dist/index.html $(EMBED_INDEX)

# Native build for the current host platform.
plugin:
	CGO_ENABLED=1 go build -buildvcs=false -tags cshared -buildmode=c-shared -o $(PLUGIN).$(EXT) $(PKG)
	rm -f $(PLUGIN).h

# Cross-compile for Linux servers (requires a C cross toolchain for arm64).
build-linux-amd64: web
	mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -buildvcs=false -tags cshared -buildmode=c-shared -o $(DIST)/$(PLUGIN)_linux_amd64.so $(PKG)

build-linux-arm64: web
	mkdir -p $(DIST)
	GOOS=linux GOARCH=arm64 CGO_ENABLED=1 go build -buildvcs=false -tags cshared -buildmode=c-shared -o $(DIST)/$(PLUGIN)_linux_arm64.so $(PKG)

clean:
	rm -f $(PLUGIN).so $(PLUGIN).dylib $(PLUGIN).h
	rm -rf $(DIST) $(WEB)/dist
