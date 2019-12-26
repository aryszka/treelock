GOSOURCE = $(shell find . -name "*.go")

default: build

.PHONY: build
build:
	go build

.PHONY: check
check:
	go test -count 1

.PHONY: .cover
.cover:
	go test -count 1 -coverprofile .cover

.PHONY: showcover
showcover: .cover
	go tool cover -html .cover

.PHONY: imports
imports:
	goimports -w $(GOSOURCE)

.PHONY: fmt
fmt:
	gofmt -w -s $(GOSOURCE)
