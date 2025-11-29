.PHONY: build test lint fmt clean install tidy

build:
	go build -o bin/cafs ./cmd/cafs

install:
	go install ./cmd/cafs

test:
	go test ./...

lint:
	golangci-lint run ./...

fmt:
	gofmt -w .
	goimports -w . 2>/dev/null || true

clean:
	rm -rf bin/ dist/

tidy:
	go mod tidy
	@for d in examples/*/; do (cd "$$d" && go mod tidy); done
