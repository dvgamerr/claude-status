.PHONY: test vet build build-pi package clean

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -trimpath -o bin/claude-status ./cmd/claude-status

build-pi:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o bin/claude-status-linux-arm64 ./cmd/claude-status

package:
	bash scripts/package.sh $${VERSION:?set VERSION, for example VERSION=v0.1.0}

clean:
	go clean
