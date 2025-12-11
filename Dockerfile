# FROM golang:1.23.6-bullseye AS builder
FROM golang:1.25.5-bookworm AS builder


WORKDIR /go/src/github.com/newrelic/newrelic-fluent-bit-output

COPY Makefile go.* *.go /go/src/github.com/newrelic/newrelic-fluent-bit-output/
COPY config/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/config
COPY metrics/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/metrics
COPY nrclient/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/nrclient
COPY record/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/record
COPY utils/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/utils

ENV SOURCE=docker

# Install cross-compilation toolchains required by the Makefile for ARM builds.
# The 'bookworm' base uses standard Debian naming conventions for these packages.
RUN apt-get update && \
    apt-get install -y \
    gcc-aarch64-linux-gnu \
    g++-aarch64-linux-gnu \
    gcc-arm-linux-gnueabihf \
    g++-arm-linux-gnueabihf \
    # Add other necessary tools like 'make' if not already in the Golang image
    && apt-get clean && rm -rf /var/lib/apt/lists/*


# Not using default value here due to this: https://github.com/docker/buildx/issues/510
ARG TARGETPLATFORM
ENV TARGETPLATFORM=${TARGETPLATFORM:-linux/amd64}
RUN echo "Building for ${TARGETPLATFORM} architecture"
RUN make ${TARGETPLATFORM}

FROM fluent/fluent-bit:4.2.0
# Expose this env variable so that the version can be used in the helm chart
ENV FBVERSION=4.2.0

COPY --from=builder /go/src/github.com/newrelic/newrelic-fluent-bit-output/out_newrelic-linux-*.so /fluent-bit/bin/out_newrelic.so
COPY *.conf /fluent-bit/etc/

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf", "-e", "/fluent-bit/bin/out_newrelic.so"]
