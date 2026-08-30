.PHONY: build build-linux run tidy vet clean

# 版本注入：make build VERSION=v1.2.3；默认取源码 internal/version 常量
VERSION ?= 
LDFLAGS := $(if $(VERSION),-X datacenter/internal/version.Version=$(VERSION),)

build:
	go build $(if $(LDFLAGS),-ldflags "$(LDFLAGS)") -o bin/server ./cmd/server

# Linux 交叉编译（发布到服务器用，见 部署指南.md）
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(if $(LDFLAGS),-ldflags "$(LDFLAGS)") -o bin/server-linux-amd64 ./cmd/server
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/scraper-linux-amd64 ./cmd/scraper

run: build
	./bin/server -config config/config.yaml

tidy:
	go mod tidy

vet:
	go vet ./...

clean:
	rm -rf bin
