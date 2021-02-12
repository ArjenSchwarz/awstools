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

.PHONY: clean
clean:
	rm -rf ./dist ./pkg

.PHONY: compile
compile:
	GOOS=linux GOARCH=amd64 go build -o pkg/linux_amd64/awstools main.go
	GOOS=darwin GOARCH=amd64 go build -o pkg/darwin_amd64/awstools main.go
	GOOS=windows GOARCH=amd64 go build -o pkg/windows_amd64/awstools.exe main.go
	GOOS=linux GOARCH=386 go build -o pkg/linux_386/awstools main.go
	GOOS=windows GOARCH=386 go build -o pkg/windows_386/awstools.exe main.go

.PHONY: package
package:
	mkdir -p dist
	zip -j dist/awstools_darwin_amd64.zip pkg/darwin_amd64/awstools
	zip -j dist/awstools_windows_amd64.zip pkg/windows_amd64/awstools.exe
	zip -j dist/awstools_windows_386.zip pkg/windows_386/awstools.exe
	tar czf dist/awstools_linux_amd64.tgz -C pkg/linux_amd64 awstools
	tar czf dist/awstools_linux_386.tgz -C pkg/linux_386 awstools

.PHONY: local
local:
	golint ./...
	go test ./...
	go build