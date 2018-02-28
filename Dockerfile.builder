FROM registry.gear.ge.com/predix_edge/golang-amd64:1.8

RUN set -ex \
	&& mkdir -p /mnt/data \
	&& chmod o+rwx /mnt/data \
        && apk add --no-cache \
               bash \
               curl \
               gcc \	
               git \
	       musl-dev
