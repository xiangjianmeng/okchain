
# -------------------------------------------------------------
# This makefile defines the following targets
#
#   - all (default) - builds all targets and runs all tests/checks
#   - gotools - installs go tools like golint
#   - images[-clean] - ensures all docker images are available[/cleaned]
#   - okchaind-image[-clean] - ensures the okchaind-image is available[/cleaned] (for behave, etc)
#   - protos - generate all protobuf artifacts based on .proto files
#   - clean - cleans the build area
#   - dist-clean - superset of 'clean' that also removes persistent state

PROJECT_NAME   = okchain/testnet
BASE_VERSION   = 0.1-preview
IS_RELEASE     = false

ifneq ($(IS_RELEASE),true)
EXTRA_VERSION ?= snapshot-$(shell git rev-parse --short HEAD)
PROJECT_VERSION=$(BASE_VERSION)-$(EXTRA_VERSION)
else
PROJECT_VERSION=$(BASE_VERSION)
endif

DOCKER_TAG=$(shell uname -m)-$(PROJECT_VERSION)

PKGNAME = github.com/$(PROJECT_NAME)
GO_LDFLAGS = -X github.com/ok-chain/okchain/metadata.Version=$(PROJECT_VERSION)
CGO_FLAGS = CGO_CFLAGS=" " CGO_LDFLAGS="-lrocksdb -lstdc++ -lm -lz -lbz2 -lsnappy"
UID = $(shell id -u)

EXECUTABLES = go docker git curl
K := $(foreach exec,$(EXECUTABLES),\
	$(if $(shell which $(exec)),some string,$(error "No $(exec) in PATH: Check dependencies")))

# SUBDIRS are components that have their own Makefiles that we can invoke
SUBDIRS = gotools sdk/node
SUBDIRS:=$(strip $(SUBDIRS))

# Make our baseimage depend on any changes to images/base or scripts/provision
BASEIMAGE_RELEASE = $(shell cat ./images/base/release)
BASEIMAGE_DEPS    = $(shell git ls-files images/base scripts/provision)

PROJECT_FILES = $(shell git ls-files)
IMAGES = base src okchaind

all: okchaind

.PHONY: $(SUBDIRS)
$(SUBDIRS):
	cd $@ && $(MAKE)

.PHONY: okchaind
okchaind-image: build/image/okchaind/.dummy


.PHONY: images
images: $(patsubst %,build/image/%/.dummy, $(IMAGES))


# We (re)build protoc-gen-go from within docker context so that
# we may later inject the binary into a different docker environment
# This is necessary since we cannot guarantee that binaries built
# on the host natively will be compatible with the docker env.
%/bin/protoc-gen-go: build/image/base/.dummy Makefile
	@echo "Building $@"
	@mkdir -p $(@D)
	@docker run -i \
		--user=$(UID) \
		-v $(abspath vendor/github.com/golang/protobuf):/opt/gopath/src/github.com/golang/protobuf \
		-v $(abspath $(@D)):/opt/gopath/bin \
		okchain/baseimage go install github.com/golang/protobuf/protoc-gen-go

#                Mark build/docker/bin artifacts explicitly as secondary
#                since they are never referred to directly. This prevents
#                the makefile from deleting them inadvertently.
.SECONDARY: build/docker/bin/okchaind build/docker/bin/okchaincli

# We (re)build a package within a docker context but persist the $GOPATH/pkg
# directory so that subsequent builds are faster
build/docker/bin/%: build/image/src/.dummy $(PROJECT_FILES)
	$(eval TARGET = ${patsubst build/docker/bin/%,%,${@}})
	@echo "Building $@"
	@mkdir -p build/docker/bin build/docker/pkg
	@docker run -i \
		--user=$(UID) \
		-v $(abspath build/docker/bin):/opt/gopath/bin \
		-v $(abspath build/docker/pkg):/opt/gopath/pkg \
		okchain/testnet-src:$(DOCKER_TAG) go install -ldflags "$(GO_LDFLAGS)" github.com/ok-chain/okchain/cmd/$(TARGET)
	@touch $@

build/bin:
	mkdir -p $@

okchaind-image: build/image/okchaind/.dummy

build/image/okchaind/.dummy: build/image/src/.dummy

build/bin/%: build/image/base/.dummy $(PROJECT_FILES)
	@mkdir -p $(@D)
	@echo "$@"
	$(CGO_FLAGS) GOBIN=$(abspath $(@D)) go install -ldflags "$(GO_LDFLAGS)" $(PKGNAME)/$(@F)
	@echo "Binary available as $@"
	@touch $@

# Special override for base-image.
build/image/base/.dummy: $(BASEIMAGE_DEPS)
	@echo "Building docker base-image"
	@mkdir -p $(@D)
	@./scripts/provision/build_baseimage.sh $(BASEIMAGE_RELEASE)
	@touch $@

okchain-base: build/image/base/.dummy

# Special override for src-image
build/image/src/.dummy: build/image/base/.dummy $(PROJECT_FILES)
	echo "=========================="
	echo "build/image/src/.dummy $@"
	echo "=========================="
	@echo "Building docker src-image start"
	mkdir -p $(@D)
	cat images/src/Dockerfile.in \
		| sed -e 's/_TAG_/$(DOCKER_TAG)/g' \
		> $(@D)/Dockerfile
	@git ls-files | tar -jcT - > $(@D)/gopath.tar.bz2
	docker build -t $(PROJECT_NAME)-src $(@D)
	docker tag $(PROJECT_NAME)-src $(PROJECT_NAME)-src:$(DOCKER_TAG)
	@touch $@
	@echo "Building docker src-image done"

# Default rule for image creation
build/image/%/.dummy: build/image/src/.dummy build/docker/bin/%
	@echo "=========================="
	@echo "build/image/%/.dummy $@"
	@echo "=========================="
	$(eval TARGET = ${patsubst build/image/%/.dummy,%,${@}})
	@echo "Building docker $(TARGET)-image"
	@mkdir -p $(@D)/bin
	@cat images/app/Dockerfile.in \
		| sed -e 's/_TAG_/$(DOCKER_TAG)/g' \
		> $(@D)/Dockerfile
	cp build/docker/bin/* $(@D)/bin
	docker build -t $(PROJECT_NAME)-$(TARGET) $(@D)
	docker tag $(PROJECT_NAME)-$(TARGET) $(PROJECT_NAME)-$(TARGET):$(DOCKER_TAG)
	@touch $@
	@echo "=========================="
	@echo "build/image/%/.dummy $@ done"
	@echo "=========================="

.PHONY: protos
protos: gotools
	./devenv/compile_protos.sh

base-image-clean:
	-docker rmi -f $(PROJECT_NAME)-baseimage
	-@rm -rf build/image/base ||:

src-image-clean: okchaind-image-clean

%-image-clean:
	$(eval TARGET = ${patsubst %-image-clean,%,${@}})
	-docker images -q $(PROJECT_NAME)-$(TARGET) | xargs -r docker rmi -f
	-@rm -rf build/image/$(TARGET) ||:

images-clean: $(patsubst %,%-image-clean, $(IMAGES))

.PHONY: $(SUBDIRS:=-clean)
$(SUBDIRS:=-clean):
	cd $(patsubst %-clean,%,$@) && $(MAKE) clean

.PHONY: clean
clean: images-clean $(filter-out gotools-clean, $(SUBDIRS:=-clean))
	-@rm -rf build ||:

.PHONY: dist-clean
dist-clean: clean gotools-clean
	-@rm -rf /var/okchian/* ||:
