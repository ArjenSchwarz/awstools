.PHONY: all
build: test clean compile

.PHONY: deps
deps:
	go get -v ./...

.PHONY: test
test:
	go get -u golang.org/x/lint/golint
	$(GOPATH)/bin/golint ./...
	go test ./...

.PHONY: local
local:
	golint ./...
	go test ./...
	go build

.PHONY: shapes
shapes:
	cd drawio/shapes && ./convert.sh