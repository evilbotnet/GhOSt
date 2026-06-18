# GhOSt top-level entry points
SHELL := /bin/bash

.PHONY: dev run-dist shell daemon release deploy-vm clean

# Inner loop on macOS: vite dev server + ghostd daemon
dev:
	./scripts/dev.sh

# Production-like smoke test: ghostd serving the built static shell on one
# origin (http://127.0.0.1:7700), exactly as it runs on the Pi. No Vite, no HMR.
run-dist: shell daemon
	@test -f .ghost-dev-token || head -c 32 /dev/urandom | xxd -p -c 64 > .ghost-dev-token
	@echo "GhOSt serving the built shell on http://127.0.0.1:7700"
	daemon/ghostd --listen 127.0.0.1:7700 --static apps/shell/dist --token-file .ghost-dev-token --dev

shell:
	pnpm --filter @ghostos/shell build

daemon:
	cd daemon && go build -o ghostd ./cmd/ghostd

# Cross-compile daemon + build shell into dist/ for Linux arm64 targets
release:
	./scripts/build-release.sh

# Push a release build into the dev VM over SSH (see os/vm/README.md)
deploy-vm:
	./scripts/deploy-vm.sh

clean:
	rm -rf apps/shell/dist daemon/ghostd dist
