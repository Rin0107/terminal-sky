.PHONY: verify test lint guards build
export CGO_ENABLED ?= 0

verify: lint test guards build

lint:
	@test -z "$$(gofmt -l .)" || (echo "gofmt が必要:"; gofmt -l .; exit 1)
	go vet ./...
	@if command -v golangci-lint >/dev/null; then golangci-lint run; else echo "skip: golangci-lint (CI で動く)"; fi

test:
	go test ./...

guards:
	node scripts/go-guard.mjs
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...

build:
	go build -trimpath -o /dev/null ./cmd/...
