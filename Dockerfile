FROM golang:1.13.5-alpine3.11 AS builder

RUN mkdir -p /go/src/envoy_nodeport_eds

WORKDIR /go/src/envoy_nodeport_eds

COPY . .

RUN CGO_ENABLED=0 go build -mod=vendor -a -installsuffix cgo -o envoy_nodeport_eds .

FROM alpine:3.11

EXPOSE 8000

RUN addgroup -S app && adduser -D -s /bin/false -G app app

COPY --from=builder /go/src/envoy_nodeport_eds/envoy_nodeport_eds /envoy_nodeport_eds

RUN chown app:app /envoy_nodeport_eds && \
  chmod +x /envoy_nodeport_eds

USER app

CMD ["/envoy_nodeport_eds"]
