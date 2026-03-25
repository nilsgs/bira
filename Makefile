BINARY  = bira
SRC     = src
VERSION = $(shell cat VERSION)
COMMIT  = $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS = -s -w -X bira/cmd.version=$(VERSION) -X bira/cmd.commit=$(COMMIT)

.PHONY: build install cross clean

build:
	cd $(SRC) && go build -ldflags "$(LDFLAGS)" -o ../$(BINARY).exe .

install:
	cd $(SRC) && go install -ldflags "$(LDFLAGS)" .

cross:
	cd $(SRC) && GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ../dist/$(BINARY)-linux-amd64 .
	cd $(SRC) && GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o ../dist/$(BINARY)-linux-arm64 .
	cd $(SRC) && GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ../dist/$(BINARY)-darwin-amd64 .
	cd $(SRC) && GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o ../dist/$(BINARY)-darwin-arm64 .
	cd $(SRC) && GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o ../dist/$(BINARY)-windows-amd64.exe .
	cd $(SRC) && GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o ../dist/$(BINARY)-windows-arm64.exe .

clean:
	rm -f $(BINARY).exe
	rm -rf dist/
