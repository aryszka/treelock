SOURCE = $(shell find . -name "*.go")

default: build

build: $(SOURCE)
	go build

check:
	go test -short

checkfull:
	go test -v -count 1 -race

.cover: $(SOURCE)
	go test -count 1 -coverprofile .cover -short

cover: .cover
	go tool cover -func .cover

showcover: .cover
	go tool cover -html .cover

imports:
	goimports -w $(SOURCE)

fmt:
	gofmt -w -s $(SOURCE)
