# Build tools
#
# Targets (see each target for more information):
#   build:	builds binaries for specified architecture
#   image:	builds the docker image
#   test:	runs lint, unit tests etc.
#   scan:	runs static analysis tools
#   clean:	removes build artifacts and images
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
NAME = cappsd

# This repo's root import path (under GOPATH)
PKG := github.build.ge.com/PredixEdgeOS/container-app-service

# Where to push the docker image.
REGISTRY ?= registry.gear.ge.com/predix_edge

# Which architecture to build - see $(ALL_ARCH) for options.
ARCH ?= amd64

# This version-strategy uses git tags to set the version string
#VERSION := $(shell git describe --tags --always --dirty)
VERSION := 1.0.0

# Git commit
GITCOMMIT := $(shell git rev-parse HEAD)

# Modules and app/service
APP := agent
SUBMODULES := config handlers provider types utils
TESTMODULES := config handlers provider utils

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
DOCKER_GID := $(shell USER_ARG="`cat /etc/group | grep docker | cut -d':' -f3`"; echo "$$USER_ARG")
DOCKER_USER := $(shell if [ "$$OSTYPE" != "darwin"* ]; then USER_ARG="--user=`id -u`:$(DOCKER_GID)"; fi; echo "$$USER_ARG")
# DOCKER_USER := $(shell if [ "$$OSTYPE" != "darwin"* ]; then USER_ARG="--user=`id -u`"; fi; echo "$$USER_ARG")
PROXY_ARGS := $(shell if [ "$$http_proxy" != "" ]; then echo "-e http_proxy=$$http_proxy"; fi)
PROXY_ARGS += $(shell if [ "$$https_proxy" != "" ]; then echo " -e https_proxy=$$https_proxy"; fi)
PROXY_ARGS += $(shell if [ "$$no_proxy" != "" ]; then echo " -e no_proxy=$$no_proxy"; fi)

INSTALL_DEPS = 1

ALL_ARCH := amd64 arm arm64

IMGARCH=$(ARCH)
ifeq ($(ARCH),amd64)
	BASEIMAGE?=registry.gear.ge.com/predix_edge/alpine-amd64:3.4
endif
ifeq ($(ARCH),arm)
	BASEIMAGE?=registry.gear.ge.com/predix_edge/alpine-arm:3.4
endif
ifeq ($(ARCH),arm64)
	BASEIMAGE?=registry.gear.ge.com/predix_edge/alpine-aarch64:3.5
	IMGARCH=aarch64
endif

IMAGE := $(REGISTRY)/$(NAME)-$(ARCH)

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

build-dirs:
	@mkdir -p bin/$(ARCH)
	@mkdir -p .go/src/$(PKG) .go/pkg .go/bin .go/std/$(ARCH)

.builder-$(ARCH):
	@echo "creating builder image ... "
	@sed \
		-e 's|#{ARCH}|$(IMGARCH)|g' \
		Dockerfile.builder > .builder-$(ARCH)
	@bash -xc "trap 'rm .builder-$(ARCH)' ERR;                              \
		docker build                                                       \
		-t $(NAME)-$(ARCH):builder                                         \
		-f .builder-$(ARCH)                                                \
		$$(echo "$(PROXY_ARGS)" | sed s/-e/--build-arg/g)                    \
		.                                                                  \
		"

fetch-deps: build-dirs .builder-$(ARCH)
	@echo "populating local .go tree ... "
	@docker run                                                            \
		--rm                                                               \
		-t                                                                 \
		$(DOCKER_USER)                                                     \
		$(PROXY_ARGS)                                                      \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-v $$(pwd)/bin/$(ARCH):/go/bin                                     \
		-v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
		-w /go/src/$(PKG)                                                  \
		$(NAME)-$(ARCH):builder                                            \
		/bin/sh -c "                                                       \
			INSTALL_DEPS=$(INSTALL_DEPS)                                   \
			./scripts/fetch-deps.sh                                        \
		"
	@echo "fetch-deps: Build Success"

bin/$(ARCH)/$(NAME): fetch-deps
	@echo "building: $@"
	@echo $(DOCKER_USER)
	@docker run                                                            \
		--rm                                                               \
		-t                                                                 \
		$(DOCKER_USER)                                                     \
		$(PROXY_ARGS)                                                      \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-v $$(pwd)/bin/$(ARCH):/go/bin                                     \
		-v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
		-w /go/src/$(PKG)                                                  \
		$(NAME)-$(ARCH):builder                                            \
		/bin/sh -c "                                                       \
			ARCH=$(ARCH)                                                   \
			VERSION=$(VERSION)                                             \
			PKG=$(PKG)                                                     \
			NAME=$(NAME)                                                   \
			GITCOMMIT=$(GITCOMMIT)                                         \
			./scripts/build.sh $(APP)                                      \
		"

scan: fetch-deps
	@echo "running static scan checks: $(ARCH)"
	@docker run                                                            \
		--rm                                                               \
		-t                                                                 \
		$(DOCKER_USER)                                                     \
		$(PROXY_ARGS)                                                      \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-v $$(pwd)/bin/$(ARCH):/go/bin                                     \
		-v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
		-w /go/src/$(PKG)                                                  \
		$(NAME)-$(ARCH):builder                                            \
		/bin/sh -c "                                                       \
			ARCH=$(ARCH)                                                   \
			VERSION=$(VERSION)                                             \
			PKG=$(PKG)                                                     \
			./scripts/scan.sh $(SUBMODULES)                                \
		"

test: fetch-deps
	@echo "running tests: $(ARCH)"
	@docker run                                                            \
		--rm                                                               \
		--privileged                                                       \
		-t                                                                 \
		$(DOCKER_USER)                                                     \
		$(PROXY_ARGS)                                                      \
		-v /var/run/docker.sock:/var/run/docker.sock                       \
		-v /var/run/docker.pid:/var/run/docker.pid                         \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-v $$(pwd)/bin/$(ARCH):/go/bin                                     \
		-v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
		-w /go/src/$(PKG)                                                  \
		$(NAME)-$(ARCH):builder                                            \
		/bin/sh -c "                                                       \
			ARCH=$(ARCH)                                                   \
			VERSION=$(VERSION)                                             \
			PKG=$(PKG)                                                     \
			./scripts/test.sh $(TESTMODULES)                               \
		"

build-shell: fetch-deps
	@echo "Entering build shell..."
	@echo $(DOCKER_USER)
	@docker run                                                            \
		-it                                                                \
		$(DOCKER_USER)                                                     \
		$(PROXY_ARGS)                                                      \
		-v $$(pwd)/.go:/go                                                 \
		-v $$(pwd):/go/src/$(PKG)                                          \
		-v $$(pwd)/bin/$(ARCH):/go/bin                                     \
		-v $$(pwd)/.go/std/$(ARCH):/usr/local/go/pkg/linux_$(ARCH)_static  \
		-w /go/src/$(PKG)                                                  \
		$(NAME)-$(ARCH):builder                                            \
		/bin/bash

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
	@if [ $(shell docker images | grep $(NAME)-$(ARCH) | wc -l) != 0 ]; then \
		docker images | grep $(NAME)-$(ARCH) | awk '{print $$3}' | xargs docker rmi -f || true; \
	fi
	rm -rf .image-* .dockerfile-* .push-* .builder-*

bin-clean:
	rm -rf .go bin
