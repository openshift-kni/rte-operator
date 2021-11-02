# Build the manager binary
FROM golang:1.17 as builder

WORKDIR /go/src/github.com/openshift-kni/rte-operator
COPY . .

# Build
RUN make binary-all

FROM registry.access.redhat.com/ubi8/ubi-minimal
COPY --from=builder /go/src/github.com/openshift-kni/rte-operator/bin/manager /bin/rte-operator
# bundle the operand, and use a backward compatible name for RTE
COPY --from=builder /go/src/github.com/openshift-kni/rte-operator/bin/exporter /bin/resource-topology-exporter
RUN mkdir /etc/resource-topology-exporter/ && \
    touch /etc/resource-topology-exporter/config.yaml
RUN microdnf install pciutils
USER 65532:65532
ENTRYPOINT ["/bin/rte-operator"]
