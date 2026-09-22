GO ?= go
BINARY ?= ohk

.PHONY: fmt lint build clean

fmt:
	$(GO) fmt ./...

lint:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "gofmt: files above need formatting"; exit 1; }
	$(GO) vet ./...

build:
	$(GO) build -o $(BINARY) .

clean:
	rm -f $(BINARY)