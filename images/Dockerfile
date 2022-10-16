# This Dockerfile is used to build the image available on DockerHub
FROM --platform=$BUILDPLATFORM golang:1.18.1 as build

ADD . /usr/src/cni-plugins
RUN  cd /usr/src/cni-plugins && make build

FROM alpine:latest
LABEL org.opencontainers.image.source https://github.com/spidernet-io/cni-plugins
COPY --from=build /usr/src/cni-plugins/.tmp/bin/ /home/cnibin
WORKDIR /
ADD ./images/entrypoint.sh /

ENTRYPOINT ["/entrypoint.sh"]

