.DEFAULT_GOAL := help

GO ?= go
APP ?= arbor
CMD_DIR ?= ./cmd/arbor
BIN_DIR ?= bin
DIST_DIR ?= dist
COMPLETIONS_DIR ?= completions
GOCACHE ?= $(CURDIR)/.gocache

VERSION ?= dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X arbor/internal/version.Number=$(VERSION) -X arbor/internal/version.Commit=$(COMMIT) -X arbor/internal/version.Date=$(DATE)

TARGETS := \
	darwin-arm64 \
	darwin-amd64 \
	linux-amd64

.PHONY: \
	help \
	deps \
	download \
	fmt \
	vet \
	check \
	test \
	test-race \
	coverage \
	run \
	build \
	install \
	completions \
	clean \
	clean-bin \
	clean-dist \
	clean-completions \
	dist \
	dist-darwin-arm64 \
	dist-darwin-amd64 \
	dist-linux-amd64 \
	checksums

help:
	@printf "%s\n" \
		"Targets:" \
		"  make deps            Sync Go module dependencies" \
		"  make download        Download Go module dependencies" \
		"  make fmt             Run go fmt ./..." \
		"  make vet             Run go vet ./..." \
		"  make check           Run fmt, vet, and test" \
		"  make test            Run go test ./..." \
		"  make test-race       Run go test -race ./..." \
		"  make coverage        Write coverage.out" \
		"  make run ARGS=...    Run the CLI from source" \
		"  make build           Build ./bin/$(APP)" \
		"  make install         Install the CLI with go install" \
		"  make completions     Generate shell completions" \
		"  make dist            Build release archives in ./$(DIST_DIR)" \
		"  make clean           Remove build and test artifacts"

deps:
	GOCACHE=$(GOCACHE) $(GO) mod tidy

download:
	GOCACHE=$(GOCACHE) $(GO) mod download

fmt:
	GOCACHE=$(GOCACHE) $(GO) fmt ./...

vet:
	GOCACHE=$(GOCACHE) $(GO) vet ./...

check: fmt vet test

test:
	GOCACHE=$(GOCACHE) $(GO) test ./...

test-race:
	GOCACHE=$(GOCACHE) $(GO) test -race ./...

coverage:
	GOCACHE=$(GOCACHE) $(GO) test ./... -coverprofile=coverage.out

run:
	GOCACHE=$(GOCACHE) $(GO) run $(CMD_DIR) $(ARGS)

build: $(BIN_DIR)/$(APP)

$(BIN_DIR)/$(APP):
	mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) $(GO) build -ldflags "$(LDFLAGS)" -o $@ $(CMD_DIR)

install:
	GOCACHE=$(GOCACHE) $(GO) install -ldflags "$(LDFLAGS)" $(CMD_DIR)

completions: build
	mkdir -p $(COMPLETIONS_DIR)
	./$(BIN_DIR)/$(APP) completion bash > $(COMPLETIONS_DIR)/$(APP).bash
	./$(BIN_DIR)/$(APP) completion zsh > $(COMPLETIONS_DIR)/_$(APP)
	./$(BIN_DIR)/$(APP) completion fish > $(COMPLETIONS_DIR)/$(APP).fish
	./$(BIN_DIR)/$(APP) completion powershell > $(COMPLETIONS_DIR)/$(APP).ps1

dist: clean-dist dist-darwin-arm64 dist-darwin-amd64 dist-linux-amd64 checksums

dist-darwin-arm64:
	@$(MAKE) _dist GOOS=darwin GOARCH=arm64

dist-darwin-amd64:
	@$(MAKE) _dist GOOS=darwin GOARCH=amd64

dist-linux-amd64:
	@$(MAKE) _dist GOOS=linux GOARCH=amd64

.PHONY: _dist
_dist: completions
	mkdir -p $(DIST_DIR)/$(APP)_$(VERSION)_$(GOOS)_$(GOARCH)/completions
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) GOCACHE=$(GOCACHE) $(GO) build \
		-trimpath \
		-ldflags "-s -w $(LDFLAGS)" \
		-o $(DIST_DIR)/$(APP)_$(VERSION)_$(GOOS)_$(GOARCH)/$(APP) \
		$(CMD_DIR)
	cp LICENSE README.md $(DIST_DIR)/$(APP)_$(VERSION)_$(GOOS)_$(GOARCH)/
	cp $(COMPLETIONS_DIR)/* $(DIST_DIR)/$(APP)_$(VERSION)_$(GOOS)_$(GOARCH)/completions/
	tar -C $(DIST_DIR)/$(APP)_$(VERSION)_$(GOOS)_$(GOARCH) -czf $(DIST_DIR)/$(APP)_$(VERSION)_$(GOOS)_$(GOARCH).tar.gz .

checksums:
	shasum -a 256 $(DIST_DIR)/*.tar.gz > $(DIST_DIR)/checksums.txt

clean: clean-bin clean-dist clean-completions
	rm -f coverage.out

clean-bin:
	rm -rf $(BIN_DIR)

clean-dist:
	rm -rf $(DIST_DIR)

clean-completions:
	rm -rf $(COMPLETIONS_DIR)
