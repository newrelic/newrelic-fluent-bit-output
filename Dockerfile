FROM golang:1.11 AS builder

WORKDIR /go/src/github.com/newrelic/newrelic-fluent-bit-output

COPY Makefile go.* *.go /go/src/github.com/newrelic/newrelic-fluent-bit-output/
COPY buffer/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/buffer
COPY config/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/config
COPY nrclient/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/nrclient
COPY utils/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/utils

ENV SOURCE docker
RUN go get github.com/fluent/fluent-bit-go/output
RUN make all

FROM fluent/fluent-bit:1.0.3

COPY --from=builder /go/src/github.com/newrelic/newrelic-fluent-bit-output/out_newrelic.so /fluent-bit/bin/
COPY *.conf /fluent-bit/etc/

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf", "-e", "/fluent-bit/bin/out_newrelic.so"]
