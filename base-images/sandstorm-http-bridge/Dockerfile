# This Dockerfile generates an image that is just the base alpine image plus
# sandstorm-http-bridge. It is not runnable as an app itself, but is probably
# a good base to build other stock images from.

FROM alpine:3.14 as builder

ENV SANDSTORM_VERSION=276

# For strip:
RUN apk add binutils

# Download the sandstorm distribution, and extract sandstorm-http-bridge from it:
RUN wget https://dl.sandstorm.io/sandstorm-${SANDSTORM_VERSION}.tar.xz
RUN tar -x sandstorm-${SANDSTORM_VERSION}/bin/sandstorm-http-bridge -f sandstorm-*.tar.xz
RUN cp sandstorm-*/bin/sandstorm-http-bridge ./
# Stripping the binary reduces its size by about 10x:
RUN strip sandstorm-http-bridge

FROM alpine:3.14
COPY --from=builder sandstorm-http-bridge /
