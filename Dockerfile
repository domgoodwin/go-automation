FROM golang:alpine as app-builder
WORKDIR /go/src/app
COPY . .
RUN apk add git

RUN CGO_ENABLED=0 go install -ldflags '-extldflags "-static"' -tags timetzdata

FROM scratch
COPY --from=app-builder /go/bin/go-automation /go-automation

COPY --from=alpine:latest /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
ENTRYPOINT ["/go-automation", "subscribe", "#"]