.PHONY: default test install
all: default test install

app=$(notdir $(shell pwd))

tool:
	go get github.com/securego/gosec/cmd/gosec

sec:
	@gosec ./...
	@echo "[OK] Go security check was completed!"

init:
	export GOPROXY=https://goproxy.cn

lint:
	#golangci-lint run --enable-all
	golangci-lint run ./...

fmt:
	gofumports -w .
	gofumpt -w .
	gofmt -s -w .
	go mod tidy
	go fmt ./...
	revive .
	goimports -w .

install: init
	go install -ldflags="-s -w" ./...
	ls -lh ~/go/bin/${app}

linux: init
	GOOS=linux GOARCH=amd64 go install -ldflags="-s -w" ./...

upx:
	ls -lh ~/go/bin/${app}
	upx ~/go/bin/${app}
	ls -lh ~/go/bin/${app}
	ls -lh ~/go/bin/linux_amd64/${app}
	upx ~/go/bin/linux_amd64/${app}
	ls -lh ~/go/bin/linux_amd64/${app}

test: init
	#go test -v ./...
	go test -v -race ./...

bench: init
	#go test -bench . ./...
	go test -tags bench -benchmem -bench . ./...

clean:
	rm coverage.out

cover:
	go test -v -race -coverpkg=./... -coverprofile=coverage.out ./...

coverview:
	go tool cover -html=coverage.out
