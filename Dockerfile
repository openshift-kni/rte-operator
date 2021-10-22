# Build the manager binary
FROM golang:1.16 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/
COPY pkg/ pkg/
COPY rte/ rte/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o exporter rte/main.go

FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /workspace/manager /bin/rte-operator
# bundle the operand, and use a backward compatible name for RTE
COPY --from=builder /workspace/exporter /bin/resource-topology-exporter
RUN mkdir /etc/resource-topology-exporter/ && \
    touch /etc/resource-topology-exporter/config.yaml
RUN microdnf install pciutils
USER 65532:65532
ENTRYPOINT ["/bin/rte-operator"]
