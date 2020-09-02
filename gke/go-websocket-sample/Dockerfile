FROM golang:1.14-alpine
ADD . /go/src/github.com/docup/docup-api/gke/go-websocket-sample
RUN cd /go/src/github.com/docup/docup-api/gke/go-websocket-sample && go build -o /go/bin/go-websocket-sample main.go

FROM alpine:latest
COPY --from=0 /go/bin/go-websocket-sample .
ENV PORT 8080
CMD ["./go-websocket-sample"]
