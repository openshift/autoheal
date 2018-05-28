FROM golang:1.10 as builder
WORKDIR /go/src/github.com/openshift/autoheal/
COPY . .
RUN hack/build-go.sh

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /go/src/github.com/openshift/autoheal/_output/local/bin/linux/amd64/autoheal .
CMD ["./autoheal"]
