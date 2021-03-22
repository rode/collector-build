.PHONY: test fmtcheck vet fmt
GOFMT_FILES?=$$(find . -name '*.go')

GO111MODULE=on

fmtcheck:
	lineCount=$(shell gofmt -l -s $(GOFMT_FILES) | wc -l | tr -d ' ') && exit $$lineCount

fmt:
	gofmt -w -s $(GOFMT_FILES)

vet:
	go vet ./...

test: fmtcheck vet
	go test -v ./... -coverprofile=coverage.txt -covermode atomic
