# This Dockerfile is used to build the image available on DockerHub
FROM --platform=$BUILDPLATFORM golang:1.18.1 as build

ARG TARGETARCH
ARG TARGETOS
ARG TARGETPLATFORM

ADD . /usr/src/cni-plugins
RUN  cd /usr/src/cni-plugins && GOOS=$TARGETOS GOARCH=$TARGETARCH make build

FROM alpine:latest
LABEL org.opencontainers.image.source https://github.com/spidernet-io/cni-plugins
COPY --from=build /usr/src/cni-plugins/.tmp/bin/ /home/cnibin
WORKDIR /
ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]
