FROM golang:1.19 AS builder


# Set the Current Working Directory inside the container
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

WORKDIR /app/cmd/ingress-trafikverket


RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /opt/diwise

LABEL "org.opencontainers.image.source"="https://github.com/diwise/ingress-trafikverket"

COPY --from=builder --chown=1001 /app/cmd/ingress-trafikverket/ingress-trafikverket /opt/diwise

RUN chown 1001 /opt/diwise
RUN chmod 700 /opt/diwise

EXPOSE 8080
USER 1001

ENTRYPOINT ["/opt/diwise/ingress-trafikverket"]
