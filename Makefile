.PHONY: build dev-server dev-web dev test lint fmt build-all

build:
	go build -o bin/recap ./cmd/recap

dev-server:
	go run ./cmd/recap serve

dev-web:
	cd web && npm run dev

dev:
	$(MAKE) -j2 dev-server dev-web

test:
	go test -race ./...

lint:
	go vet ./...

fmt:
	gofmt -w .
	goimports -w .

build-all:
	GOOS=linux GOARCH=amd64 go build -o bin/recap-linux-amd64 ./cmd/recap
	GOOS=linux GOARCH=arm64 go build -o bin/recap-linux-arm64 ./cmd/recap
	GOOS=darwin GOARCH=amd64 go build -o bin/recap-darwin-amd64 ./cmd/recap
	GOOS=darwin GOARCH=arm64 go build -o bin/recap-darwin-arm64 ./cmd/recap
	GOOS=windows GOARCH=amd64 go build -o bin/recap-windows-amd64.exe ./cmd/recap
