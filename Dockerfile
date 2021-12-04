FROM golang:1.15 AS builder
COPY . /go/src/service
WORKDIR /go/src/service/cmd
RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags='-w -s' -o /go/bin/service

FROM alpine:latest
COPY --from=builder /go/bin/service /go/bin/service
ENTRYPOINT ["go/bin/service"]


