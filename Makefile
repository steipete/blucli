.PHONY: test lint fmt

test:
	go test ./...

lint:
	golangci-lint run --timeout=5m

fmt:
	golangci-lint run --fix --timeout=5m

