FROM golang:1.27.3-bookworm AS builder

WORKDIR /go/src/github.com/newrelic/newrelic-fluent-bit-output

COPY Makefile go.* *.go /go/src/github.com/newrelic/newrelic-fluent-bit-output/
COPY config/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/config
COPY metrics/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/metrics
COPY nrclient/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/nrclient
COPY record/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/record
COPY utils/ /go/src/github.com/newrelic/newrelic-fluent-bit-output/utils

ENV SOURCE=docker

# Not using default value here due to this: https://github.com/docker/buildx/issues/510
ARG TARGETPLATFORM
ENV TARGETPLATFORM=${TARGETPLATFORM:-linux/amd64}
RUN echo "Building for ${TARGETPLATFORM} architecture"
RUN make ${TARGETPLATFORM}

# CVE-2026-111497: the fluent/fluent-bit:5.1.5 runtime image ships libbrotli1 1.1.0-2+b7,
# which has an out-of-bounds write in the Brotli decoder. Fixed in Debian trixie-security
# as 1.1.0-2+deb13u1 (DSA-6582-1), but the runtime image has no shell or package manager
# (apt/dpkg) to install the fix in place, and fluent/fluent-bit hasn't republished a base
# image with the patched package yet -- so the patched shared libraries are built in a
# throwaway Debian stage and copied in directly, overwriting the vulnerable ones.
#
# This stage is built once per target platform by buildx (linux/amd64, linux/arm64,
# linux/arm/v7), so it resolves its own multiarch lib directory (e.g. x86_64-linux-gnu vs
# aarch64-linux-gnu vs arm-linux-gnueabihf) via dpkg-architecture rather than hardcoding
# one -- everything it produces is staged under /out, mirroring the real path for
# whichever architecture is currently being built, so the final stage's COPY never needs
# to name an architecture either.
FROM debian:trixie-slim AS brotli-fix
RUN apt-get update && apt-get install -y --no-install-recommends libbrotli1 && rm -rf /var/lib/apt/lists/* \
 && libdir="$(dirname "$(dpkg -L libbrotli1 | grep -m1 '\.so')")" \
 && mkdir -p "/out${libdir}" /out/var/lib/dpkg/status.d \
 && cp -a "${libdir}"/libbrotli*.so* "/out${libdir}/" \
 && awk '/^Package: libbrotli1$/{p=1} p{print} p && /^$/{exit}' /var/lib/dpkg/status \
    | grep -v '^Status:' > /out/var/lib/dpkg/status.d/libbrotli1

FROM fluent/fluent-bit:5.1.5
# Expose this env variable so that the version can be used in the helm chart
ENV FBVERSION=5.1.5

# See the brotli-fix stage above: both paths are staged generically under /out so this
# works unmodified across all architectures the image is built for.
COPY --from=brotli-fix /out/usr/lib/ /usr/lib/
# The runtime image's dpkg database uses a non-standard per-package split layout
# (/var/lib/dpkg/status.d/<pkg>) that debian:trixie-slim's dpkg doesn't produce, so the
# fixed stage's package record can't be copied as-is; brotli-fix reformats it to match.
COPY --from=brotli-fix /out/var/lib/dpkg/status.d/libbrotli1 /var/lib/dpkg/status.d/libbrotli1
COPY --from=builder /go/src/github.com/newrelic/newrelic-fluent-bit-output/out_newrelic-linux-*.so /fluent-bit/bin/out_newrelic.so
COPY *.conf /fluent-bit/etc/

CMD ["/fluent-bit/bin/fluent-bit", "-c", "/fluent-bit/etc/fluent-bit.conf", "-e", "/fluent-bit/bin/out_newrelic.so"]
