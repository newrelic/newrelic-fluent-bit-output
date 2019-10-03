FROM golang:1.11 AS builder

WORKDIR /go/src/github.com/newrelic/fluent-bit/newrelic-fluent-bit-output

COPY .git Makefile go.* *.go *.h /go/src/github.com/newrelic/fluent-bit/newrelic-fluent-bit-output/
ENV SOURCE docker
RUN go get github.com/fluent/fluent-bit-go/output
RUN make all

FROM fluent/fluent-bit:1.0.3

COPY --from=builder /go/src/github.com/newrelic/fluent-bit/newrelic-fluent-bit-output/out_newrelic.so /fluent-bit/bin/
COPY *.conf /fluent-bit/etc/

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf", "-e", "/fluent-bit/bin/out_newrelic.so"]
