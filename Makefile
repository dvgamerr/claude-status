.PHONY: test coverage vet check build build-pi package clean

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

vet:
	go vet ./...

check: test vet build-pi

build:
	go build -trimpath -o bin/claude-status ./cmd/claude-status

build-pi:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o bin/claude-status-linux-arm64 ./cmd/claude-status

package:
	bash scripts/package.sh $${VERSION:?set VERSION, for example VERSION=v0.1.0}

clean:
	go clean
