FROM golang:1.21-alpine as buildbase

RUN apk add git build-base

WORKDIR /go/src/github.com/rarimo/tss-svc
COPY vendor .
COPY . .

ENV GO111MODULE="on"
ENV CGO_ENABLED=1
ENV GOOS="linux"

RUN GOOS=linux go build  -o /usr/local/bin/tss-svc /go/src/github.com/rarimo/tss-svc


FROM alpine:3.9

COPY --from=buildbase /usr/local/bin/tss-svc /usr/local/bin/tss-svc
RUN apk add --no-cache ca-certificates

ENTRYPOINT ["tss-svc"]
