FROM golang:1.19 AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

WORKDIR /app/cmd/integration-fimbul

RUN GOOS=linux GOARCH=amd64 go build

FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /opt/diwise

COPY --from=builder --chown=1001 /app/cmd/integration-fimbul/integration-fimbul /opt/diwise/

RUN chown 1001 /opt/diwise
RUN chmod 700 /opt/diwise

USER 1001

ENTRYPOINT ["/opt/diwise/integration-fimbul"]
