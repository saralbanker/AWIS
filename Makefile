.PHONY: build test lint race contract e1 bench release-dry pytest \
        verify gofmt-check vet oip-isolation integration

# verify is the single source of truth for "are the gates green?".
# CI runs exactly this target and nothing else, so a local PASS and a CI PASS
# mean the same thing. Ordered fast-to-slow so failures surface early.
# (Stabilization sprint S6: the previous split — V-COMMON checking four things
# and CI checking a different seven — is how gofmt drift went unnoticed across
# eight milestones.)
verify: gofmt-check vet lint oip-isolation build test race e1
	@echo "verify: ALL GATES PASSED"

gofmt-check:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then \
		echo "gofmt: files not formatted:"; echo "$$out"; exit 1; \
	fi
	@echo "gofmt: clean"

vet:
	go vet ./...
	cd apps/oip && go vet ./...

# apps/oip must be a real standalone module (IMP §15), not a go.work artifact.
# GOWORK=off is the only way to prove it: every other gate resolves the SDK
# through the workspace and cannot see a missing require directive.
oip-isolation:
	cd apps/oip && GOWORK=off go build ./...
	cd apps/oip && GOWORK=off go vet ./...

build:
	go build ./...
	cd apps/oip && go build ./...

test:
	go test ./...
	cd apps/oip && go test ./...

lint:
	golangci-lint run ./...
	cd apps/oip && golangci-lint run ./...

race:
	go test -race ./...
	cd apps/oip && go test -race ./...

contract:
	go test -v ./internal/storage/... -run 'Contract|Suite|TestSQLiteStorageContract' -count=1

e1:
	go test ./internal/engine/... -run TestE1 -count=1

integration:
	go test -tags integration -count=1 -timeout 15m ./test/integration/...

# bench runs the M18 NFR benchmark suite (IMP §27.M18; PRD §11, §31, §32). Each
# benchmark runs once (-benchtime=1x): sample counts are fixed by the harness,
# not by b.N. Each prints one result line per NFR with its threshold, measured
# percentiles, fixture size and RESULT.
#
# Exit status is non-zero only if a benchmark fails to run to completion, or an
# expected benchmark produced no result. A MISS is a measured result: it is
# reported here and fixed, if at all, only by a targeted C4 card (IMP §27.M18).
# Test binaries run one after another (-p 1), so one package's fixture I/O can
# never overlap another package's timed sections.
BENCH_EXPECT := BenchmarkNFRP01 BenchmarkNFRP02 BenchmarkNFRP04 BenchmarkNFRP05 BenchmarkNFRP06 BenchmarkNFRP07 BenchmarkNFRP08 BenchmarkNFRP09
bench:
	@out=$$(mktemp) || exit 1; \
	if ! go test -p 1 -run '^$$' -bench 'BenchmarkNFRP' -benchtime=1x -count=1 -timeout 30m ./internal/engine/ ./cmd/awis/ > "$$out" 2>&1; then \
		cat "$$out"; rm -f "$$out"; echo "bench: FAILED: a benchmark did not run to completion"; exit 1; \
	fi; \
	cat "$$out"; \
	for b in $(BENCH_EXPECT); do \
		if ! grep -q "^$$b" "$$out"; then rm -f "$$out"; echo "bench: FAILED: expected benchmark $$b produced no result"; exit 1; fi; \
	done; \
	rm -f "$$out"; \
	echo "bench: all eight NFR benchmarks ran to completion"

# release-dry proves the release packaging pipeline (IMP §21 CD) and publishes
# nothing. goreleaser runs with --snapshot: no tag, no GitHub release, no remote,
# no GitHub API call. It builds the four platform archives and checksums into
# dist/ (gitignored). The goreleaser version is pinned here and run as a Go
# module, so every machine gets the identical tool with no global install; the
# first run needs the Go module proxy, later runs use the module cache.
# The pin is the newest release whose go.mod needs no Go newer than the repo
# toolchain (go.mod: toolchain go1.26.4). v2.18.x needs go1.27 and would make
# `go run` download a second toolchain, so it is not used.
GORELEASER_VERSION := v2.17.0

release-dry:
	go run github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION) release --snapshot --clean
	@n=$$(ls dist/*.tar.gz 2>/dev/null | wc -l | tr -d ' '); \
	if [ "$$n" -ne 4 ] || [ ! -s dist/checksums.txt ]; then \
		echo "release-dry: FAILED: expected 4 platform archives and dist/checksums.txt, found $$n archive(s)"; exit 1; \
	fi; \
	echo "release-dry: $$n platform archives + checksums in dist/ (snapshot; nothing published)"

pytest:
	python3 -m pytest --rootdir=python/awis-step python/awis-step/tests -q
	python3 -m pytest --rootdir=python/awis-plugin python/awis-plugin/tests -q
	python3 -m pytest --rootdir=plugins/git-context-plugin plugins/git-context-plugin/tests -q
