FROM registry.access.redhat.com/ubi9/go-toolset:1.18 as builder

WORKDIR /opt/app-root/src
COPY . .

RUN go build -o sprayproxy main.go

FROM registry.access.redhat.com/ubi9/ubi:9.1.0
COPY --from=builder /opt/app-root/src/sprayproxy /usr/local/bin/sprayproxy
USER 1001:1001

ENTRYPOINT [ "sprayproxy", "server" ]