# win-automation tasks

default:
  @just --list

fmt:
  gofmt -w .

test:
  go test ./...

vet:
  go vet ./...

tidy:
  go mod tidy

build:
  go build ./cmd/win-automation

run *args:
  go run ./cmd/win-automation {{args}}
