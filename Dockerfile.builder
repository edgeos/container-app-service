FROM registry.gear.ge.com/predix_edge/golang-#{ARCH}:1.8

RUN set -ex \
        && apk add --no-cache \
               bash \
               curl \
               gcc \	
               git \
	       musl-dev
