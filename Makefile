GENERATED = noclist

.PHONY: all
all: build

.PHONY: test
test:
	go test ./...
	go vet ./...

.PHONY: build
build: test
	go build -o noclist cmd/noclist/main.go

.PHONY: clean
clean:
	rm -rf $(GENERATED)
