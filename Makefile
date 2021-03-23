.PHONY: test fmtcheck vet fmt licensecheck license
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

licensecheck:
	missingLicenses=$(shell addlicense -check $(LICENSE_FILES) | wc -l | tr -d ' ') && exit $$missingLicenses

license:
	addlicense -c 'The Rode Authors' $(LICENSE_FILES)

vet:
	go vet ./...

test: fmtcheck vet licensecheck
	go test -v ./... -coverprofile=coverage.txt -covermode atomic
