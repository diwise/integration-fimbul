FROM --platform=$BUILDPLATFORM docker.io/golang:1.21 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

WORKDIR /app/cmd/integration-fimbul

ARG TARGETARCH
ARG TARGETOS
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build

FROM registry.access.redhat.com/ubi9/ubi-minimal
WORKDIR /opt/diwise

COPY --from=builder --chown=1001 /app/cmd/integration-fimbul/integration-fimbul /opt/diwise/

RUN chown 1001 /opt/diwise
RUN chmod 700 /opt/diwise

USER 1001

ENTRYPOINT ["/opt/diwise/integration-fimbul"]
