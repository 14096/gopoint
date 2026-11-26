.DEFAULT_GOAL := build

.PHONY: doc build fmt fmt-check vet test race cover tidy check clean

doc:
	go run golang.org/x/pkgsite/cmd/pkgsite@latest -http=localhost:8080

build:
	go build ./...

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (gofmt -l . && echo "run 'make fmt'" && exit 1)

vet:
	go vet ./...

test:
	go test ./...

race:
	go test -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -n 1

tidy:
	go mod tidy

check: fmt-check vet race

clean:
	rm -f coverage.out
