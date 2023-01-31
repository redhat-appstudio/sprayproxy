FROM registry.access.redhat.com/ubi9/go-toolset:1.18 as builder

WORKDIR /opt/app-root/src

COPY go.mod go.mod
COPY go.sum go.sum
COPY vendor/ vendor/
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY main.go main.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o sprayproxy main.go

FROM registry.access.redhat.com/ubi9-minimal:9.1.0

COPY --from=builder /opt/app-root/src/sprayproxy /usr/local/bin/sprayproxy

USER 65532:65532

ENTRYPOINT [ "sprayproxy", "server" ]