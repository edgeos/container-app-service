FROM golang:1.11.2-alpine3.8

RUN set -ex \
	&& mkdir -p /mnt/data \
	&& chmod o+rwx /mnt/data \
        && apk add --no-cache \
               bash \
               curl \
               gcc \	
               git \
	       musl-dev
