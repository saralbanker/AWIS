.PHONY: build test lint race contract e1 bench release-dry pytest \
        verify gofmt-check vet oip-isolation

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
# On main today apps/oip is still a stub with no SDK imports, so this passes
# trivially — it is wired now so the guard is already in place when M15 lands.
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

bench:
	@echo "NOT-YET (wired at M02/M06/M18): bench"

release-dry:
	@echo "NOT-YET (wired at M02/M06/M18): release-dry"

pytest:
	python3 -m pytest --rootdir=python/awis-step python/awis-step/tests -q
	python3 -m pytest --rootdir=python/awis-plugin python/awis-plugin/tests -q
	python3 -m pytest --rootdir=plugins/git-context-plugin plugins/git-context-plugin/tests -q
