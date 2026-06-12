# GhOSt top-level entry points
SHELL := /bin/bash

.PHONY: dev shell daemon release deploy-vm clean

# Inner loop on macOS: vite dev server + ghostd daemon
dev:
	./scripts/dev.sh

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
