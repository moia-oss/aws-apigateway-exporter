FROM golang:1.25.5 AS builder
COPY ./ /exporter/
WORKDIR /exporter
RUN make build-linux

# Create minimal passwd and group files for non-root user
RUN echo "appuser:x:9999:9999::/:/sbin/nologin" > /tmp/passwd && \
    echo "appusers:x:9999:" > /tmp/group

FROM scratch

# Copy CA certificates from builder for HTTPS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy user and group files
COPY --from=builder /tmp/passwd /etc/passwd
COPY --from=builder /tmp/group /etc/group

# Copy the binary
COPY --from=builder /exporter/bin/linux_amd64/aws-apigateway-exporter \
    /bin/aws-apigateway-exporter

USER 9999:9999

EXPOSE 9389

ENTRYPOINT ["/bin/aws-apigateway-exporter"]
