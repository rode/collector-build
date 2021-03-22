# syntax = docker/dockerfile:experimental
# Build the manager binary
FROM golang:1.15-alpine as builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.sum /workspace/

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY config config

# Build
RUN --mount=type=cache,target=/root/.cache/go-build CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o collector-build

# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot as runner
WORKDIR /
COPY --from=builder /workspace/collector-build .
USER nonroot:nonroot

ENTRYPOINT ["./collector-build"]
EXPOSE 8080
