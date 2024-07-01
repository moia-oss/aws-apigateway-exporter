FROM golang:1.22.4
COPY ./ /exporter/
WORKDIR /exporter
RUN make build-linux

FROM alpine:3.20.1
RUN apk add --no-cache ca-certificates
COPY --from=0 /exporter/bin/linux_amd64/aws-apigateway-exporter \
    /bin/aws-apigateway-exporter

ENV USER=appuser
ENV GROUP=appusers
ENV UID=9999
ENV GID=9999

RUN addgroup --gid "$GID" "$GROUP" \
    && adduser \
    --disabled-password \
    --gecos "" \
    --ingroup "$GROUP" \
    --no-create-home \
    --uid "$UID" \
    "$USER"

USER "$USER"

EXPOSE 9389

ENTRYPOINT ["/bin/aws-apigateway-exporter"]
