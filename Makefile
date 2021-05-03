.PHONY: test fmtcheck vet fmt coverage license mocks run
MAKEFLAGS += --silent
GOFMT_FILES?=$$(find . -name '*.go' | grep -v proto)
LICENSE_FILES=$$(find -E . -regex '.*\.(go|proto)')

GO111MODULE=on

generate:
	docker build ./scripts/generate -t ghcr.io/rode/collector-build/generate:latest
	docker run -it --rm -v $$(pwd):/collector-build ghcr.io/rode/collector-build/generate:latest

fmtcheck:
	lineCount=$(shell gofmt -l -s $(GOFMT_FILES) | wc -l | tr -d ' ') && exit $$lineCount

fmt:
	gofmt -w -s $(GOFMT_FILES)

license:
	addlicense -c 'The Rode Authors' $(LICENSE_FILES)

mocks:
	go install github.com/maxbrunsfeld/counterfeiter/v6@v6.4.1
	COUNTERFEITER_NO_GENERATE_WARNING="true" go generate ./...

vet:
	go vet ./...

coverage: test
	go tool cover -html=coverage.txt

test: fmtcheck vet
	go test -v ./... -coverprofile=coverage.txt -covermode atomic

run:
	go run main.go --rode-host=localhost:50051 --rode-insecure --debug
