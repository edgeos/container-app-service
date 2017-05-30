# Build tools
#
# Targets (see each target for more information):
#   build:	builds binaries for specified architecture
#   image:	builds the docker image
#   test:	runs lint, unit tests etc.
#   scan:	runs static analysis tools
#   clean:	removes build artifacts and images
#   vendor:     get vendor dependencies
#   push:	pushes image to registry
#
#   all-build:	builds binaries for all target architectures
#   all-images:	builds the docker images for all target architectures
#   all-push:	pushes the images for all architectures to registry
#


###
### Customize  these variables
###

# The binary to build (just the basename).
NAME := cappsd

# This repo's root import path (under GOPATH)
PKG := github.build.ge.com/container-app-service

# Where to push the docker image.
REGISTRY ?= registry.gear.ge.com/predix_edge

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# Which OS to build (Linux or Darwin)
OS ?= linux

# This version-strategy uses git tags to set the version string
#VERSION := $(shell git describe --tags --always --dirty)
VERSION := 1.0.0

# Repo for Artifactory deployment
REPO = https://devcloud.swcoe.ge.com/artifactory/UQBMU/OS/Yocto/mirror

###
### These variables should not need tweaking.
###

# Platform specific USER  and proxy crud:
# On linux, run the container with the current uid, so files produced from
# within the container are owned by the current user, rather than root.
#
# On OSX, don't do anything with the container user, and let boot2docker manage
# permissions on the /Users mount that it sets up
DOCKER_USER := $(shell if [[ "$$OSTYPE" != "darwin"* ]]; then USER_ARG="--user=`id -u`"; fi; echo "$$USER_ARG")
PROXY_ARGS := -e http_proxy=$(http_proxy) -e https_proxy=$(https_proxy) -e no_proxy=$(no_proxy)

#SRC_DIRS := src pkg # directories which hold app source (not vendored)

ALL_ARCH := amd64 arm arm64

ifeq ($(ARCH),amd64)
#    BASEIMAGE?=registry.gear.ge.com/predix_edge/alpine
    BASEIMAGE?=docker:1.13.1
endif
ifeq ($(ARCH),arm)
    BASEIMAGE?=registry.gear.ge.com/predix_edge/alpine-armhf
endif
ifeq ($(ARCH),arm64)
    BASEIMAGE?=registry.gear.ge.com/predix_edge/alpine-arm64
endif

IMAGE := $(REGISTRY)/$(NAME)-$(ARCH)

#BUILD_IMAGE ?= registry.gear.ge.com/predix_edge/bp-golang:1.6
BUILD_IMAGE ?= golang:1.6.2-alpine

# Default target
all: build

# Builds the binary in a Docker container and copy to volume mount
build-%:
	@$(MAKE) --no-print-directory ARCH=$* build

# Builds the docker image and tags it appropriately
image-%:
	@$(MAKE) --no-print-directory ARCH=$* image

# Pushes the build docker image to the specified registry
push-%:
	@$(MAKE) --no-print-directory ARCH=$* push

# Builds all the binaries in a Docker container and copies to volume mount
all-build: $(addprefix build-, $(ALL_ARCH))

# Builds all docker images and tags them appropriately
all-image: $(addprefix image-, $(ALL_ARCH))

# Builds and pushes all images to registry
all-push: $(addprefix push-, $(ALL_ARCH))

build: bin/$(ARCH)/$(NAME)

bin/$(ARCH)/$(NAME): build-dirs
	@echo "building: $@"
	@echo $(DOCKER_USER)
	@docker run                                                            \
	    -t                                                                 \
	    $(DOCKER_USER)                                                     \
	    $(PROXY_ARGS)                                                      \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/bin/$(ARCH):/go/bin                                     \
	    -v $$(pwd)/bin/$(ARCH):/go/bin/linux_$(ARCH)                       \
	    -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    /bin/sh -c "                                                       \
	        ARCH=$(ARCH)                                                   \
			OS=$(OS)													   \
	        VERSION=$(VERSION)                                             \
			PKG=$(PKG)                                                     \
			NAME=$(NAME)                                                   \
			./build/build.sh                                               \
	    "

build-shell: build-dirs
	@echo "Entering build shell..."
	@echo $(DOCKER_USER)
	@docker run                                                            \
	    -it                                                                \
	    $(DOCKER_USER)                                                     \
	    $(PROXY_ARGS)                                                      \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/bin/$(ARCH):/go/bin                                     \
	    -v $$(pwd)/bin/$(ARCH):/go/bin/linux_$(ARCH)                       \
	    -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    /bin/sh


DOTFILE_IMAGE = $(subst /,_,$(IMAGE))-$(VERSION)
image: .image-$(DOTFILE_IMAGE) image-name
.image-$(DOTFILE_IMAGE): bin/$(ARCH)/$(NAME) Dockerfile.in
	@sed \
	    -e 's|ARG_NAME|$(NAME)|g' \
	    -e 's|ARG_ARCH|$(ARCH)|g' \
	    -e 's|ARG_FROM|$(BASEIMAGE)|g' \
	    Dockerfile.in > .dockerfile-$(ARCH)
	@docker build -t $(IMAGE):$(VERSION) -f .dockerfile-$(ARCH) .
	@docker images -q $(IMAGE):$(VERSION) > $@

image-name:
	@echo "image: $(IMAGE):$(VERSION)"

push: .push-$(DOTFILE_IMAGE) push-name
.push-$(DOTFILE_IMAGE): .image-$(DOTFILE_IMAGE)
	@gcloud docker push $(IMAGE):$(VERSION)
	@docker images -q $(IMAGE):$(VERSION) > $@

push-name:
	@echo "pushed: $(IMAGE):$(VERSION)"

version:
	@echo $(VERSION)

test: build-dirs
	@docker run                                                            \
	    -t                                                                 \
	    $(DOCKER_USER)                                                     \
	    $(PROXY_ARGS)                                                      \
	    -v $$(pwd)/.go:/go                                                 \
	    -v $$(pwd):/go/src/$(PKG)                                          \
	    -v $$(pwd)/bin/$(ARCH):/go/bin                                     \
	    -v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
	    -w /go/src/$(PKG)                                                  \
	    $(BUILD_IMAGE)                                                     \
	    /bin/sh -c "                                                       \
	        ./build/test.sh ./...                                          \
	    "

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

scan:
	@echo "Running scans"
	@echo "   TODO - scans"
	@echo "Done running scans"

#TODO: once we have specific versions of cappsd we should commit those in artifactory container-app-services
#      instead of populating latest into sources as well, need to add version number too.
deploy:
	@read -p "Enter Artifactory Username: " ART_USER; \
        printf "password: "; \
        read -s ART_PASS; \
	echo  "User: $${ART_USER}  \
		   \nArch: $(ARCH) \
		   \nRepo: $(REPO) \
		   "; \
	for f in `find ./bin/${OS} -name cappsd`; do \
	  curl -X PUT --user "$${ART_USER}:$${ART_PASS}" -T "$${f}" "$(REPO)/container-app-service/${NAME}_`echo $${f} | cut -d'/' -f4`"; \
	  curl -X PUT --user "$${ART_USER}:$${ART_PASS}" -T "$${f}" "$(REPO)/sources/${NAME}_`echo $${f} | cut -d'/' -f4`"; \
        done; \
	curl -X PUT --user "$${ART_USER}:$${ART_PASS}" -T "./ecs.json" "$(REPO)/container-app-service/ecs.json"; \
	curl -X PUT --user "$${ART_USER}:$${ART_PASS}" -T "./ecs.json" "$(REPO)/sources/ecs.json"

clean: image-clean bin-clean

image-clean:
	@if [ $(shell docker ps -a | grep $(IMAGE) | wc -l) != 0 ]; then \
		docker ps -a | grep $(IMAGE) | awk '{print $$1 }' | xargs docker rm -f; \
	fi
	@if [ $(shell docker images | grep $(IMAGE) | wc -l) != 0 ]; then \
		docker images | grep $(IMAGE) | awk '{print $$3}' | xargs docker rmi -f || true; \
	fi
	rm -rf .image-* .dockerfile-* .push-*

bin-clean:
	rm -rf .go bin
