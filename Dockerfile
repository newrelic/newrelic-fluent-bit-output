FROM golang:1.11 AS builder

WORKDIR /go/src/github.com/newrelic/newrelic-fluent-bit-output

COPY Makefile go.* *.go /go/src/github.com/newrelic/newrelic-fluent-bit-output/
COPY config/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/config
COPY nrclient/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/nrclient
COPY record/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/record
COPY utils/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/utils

ENV SOURCE docker
RUN go get github.com/fluent/fluent-bit-go/output
RUN go get github.com/sirupsen/logrus

# Not using default value here due to this: https://github.com/docker/buildx/issues/510
ARG TARGETPLATFORM
ENV TARGETPLATFORM=${TARGETPLATFORM:-linux/amd64}
RUN echo "Building for ${TARGETPLATFORM} architecture"
RUN make ${TARGETPLATFORM}

FROM fluent/fluent-bit:1.9.4

COPY --from=builder /go/src/github.com/newrelic/newrelic-fluent-bit-output/out_newrelic-linux-*.so /fluent-bit/bin/out_newrelic.so
COPY *.conf /fluent-bit/etc/

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf", "-e", "/fluent-bit/bin/out_newrelic.so"]
