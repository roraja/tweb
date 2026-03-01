.PHONY: build run clean build-all

GO := GO111MODULE=on go

build:
	$(GO) build -o tweb .

run:
	$(GO) run .

clean:
	rm -f tweb tweb-*

build-all:
	GOOS=linux   GOARCH=amd64 $(GO) build -o tweb-linux-amd64 .
	GOOS=linux   GOARCH=arm64 $(GO) build -o tweb-linux-arm64 .
	GOOS=darwin  GOARCH=amd64 $(GO) build -o tweb-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64 $(GO) build -o tweb-darwin-arm64 .
	GOOS=windows GOARCH=amd64 $(GO) build -o tweb-windows-amd64.exe .
