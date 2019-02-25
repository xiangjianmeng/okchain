NAME=okchain/testnet-baseimage
VERSION=$(shell cat ./release)
ARCH=$(shell uname -m)
DOCKER_TAG ?= $(ARCH)-$(VERSION)
VAGRANTIMAGE=packer_virtualbox-iso_virtualbox.box

DOCKER_BASE_x86_64=ubuntu:xenial
DOCKER_BASE_s390x=s390x/ubuntu:xenial
DOCKER_BASE_ppc64le=ppc64le/ubuntu:xenial

DOCKER_BASE=$(DOCKER_BASE_$(ARCH))

ifeq ($(DOCKER_BASE), )
$(error "Architecture \"$(ARCH)\" is unsupported")
endif


# strips off the post-processors that try to upload artifacts to the cloud
packer-local.json: packer.json
	jq 'del(."post-processors"[0][1]) | del(."post-processors"[1][1])' packer.json > $@

all: docker


Dockerfile: Dockerfile.in Makefile
	@echo "# Generated from Dockerfile.in.  DO NOT EDIT!" > $@
	@cat Dockerfile.in | \
	sed -e  "s|_DOCKER_BASE_|$(DOCKER_BASE)|" >> $@

docker: Dockerfile release
	@echo "Generating docker"
	@docker build -t $(NAME):$(DOCKER_TAG) .

push:
	@echo "You will need your ATLAS_TOKEN set for this to succeed"
	packer push -name $(NAME) packer.json

remove:
	-vagrant box remove --box-version 0 $(NAME)

clean: remove
	-rm $(VAGRANTIMAGE)
	-rm Dockerfile
	-rm packer-local.json
