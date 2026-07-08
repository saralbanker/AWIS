.PHONY: build test lint race contract e1 bench release-dry

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
