#!/usr/bin/make -f

# Common package information.
include pkg-info

DESCRIPTION := $(shell git describe --tags --match 'v*' | cut -d'v' -f2-)
VERSION ?= $(shell echo $(DESCRIPTION) | cut -d'-' -f1)
RELEASE ?= $(shell echo $(DESCRIPTION) | cut -d'-' -f2- -s | tr '-' '.')
ARCH ?= $(shell go env GOARCH)

ifeq ($(VERSION),)
$(error No VERSION available, please set manually.)
endif
ifeq ($(RELEASE),)
RELEASE := 1
endif

ifeq ($(ARCH),amd64)
DEB_ARCH := amd64
RPM_ARCH := x86_64
KERNEL_ARCH := x86
else
    ifeq ($(ARCH),386)
DEB_ARCH := i386
RPM_ARCH := i386
KERNEL_ARCH := x86
    else
$(error Unknown arch "$(ARCH)".)
    endif
endif

ifeq ($(KERNEL),y)
# Figure out our kernel path.
# You can override all these settings, but
# nothing will be added unless you pass KERNEL=y.
KERNEL_RELEASE ?= $(shell uname -r)
KERNEL_SOURCE ?= /lib/modules/$(KERNEL_RELEASE)/build
KERNEL_INCLUDE ?= $(KERNEL_SOURCE)/include/uapi
KERNEL_ARCH_INCLUDE ?= $(KERNEL_SOURCE)/arch/$(KERNEL_ARCH)/include/uapi
CGO_CFLAGS ?= -I$(KERNEL_INCLUDE) -I$(KERNEL_ARCH_INCLUDE)
endif

# Our default target.
all: dist
.PHONY: all

# Our GO source build command.
# This includes our kernel include path (if set).
go_build = @GOPATH=$(CURDIR) CGO_CFLAGS="$(CGO_CFLAGS)" $(1)

ARCH ?= $(shell arch)
go-build: go-fmt go-test
go-install: go-fmt go-test
go-%:
	$(call go_build,go $* novmm)
	$(call go_build,go $* noguest)
go-bench:
	$(call go_build,go test -bench=".*" novmm)
	$(call go_build,go test -bench=".*" noguest)
go-fmt:
	$(call go_build,gofmt -l=true -w=true src/novmm/$*)
	$(call go_build,gofmt -l=true -w=true src/noguest/$*)

test: go-test
.PHONY: test

fmt: go-fmt
.PHONY: fmt

clean:
	@rm -rf bin/ pkg/ dist/ doc/ _obj
	@rm -rf debbuild/ rpmbuild/ *.deb *.rpm
.PHONY: clean

dist:
	@# Build the tools.
	@$(MAKE) clean && $(MAKE) go-install
	@# Install our scripts.
	@mkdir -p dist/usr/bin
	@install -m 0755 scripts/* dist/usr/bin
	@# Install our libexec directory.
	@rm -rf dist/lib && cp -ar lib/ dist/usr/lib
	@install -m 0755 bin/* dist/usr/lib/novm/libexec
	@# Install our python code.
	@mkdir -p dist/usr/lib/novm/python
	@rsync -ru --delete --exclude=*.pyc novm \
	    dist/usr/lib/novm/python/

.PHONY: dist

deb: dist
	@rm -rf debbuild && mkdir -p debbuild
	@rsync -ruav packagers/DEBIAN debbuild
	@rsync -ruav dist/ debbuild
	@chmod 755 debbuild/DEBIAN
	@sed -i "s/VERSION/$(VERSION)-$(RELEASE)/" debbuild/DEBIAN/control
	@sed -i "s/MAINTAINER/$(MAINTAINER)/" debbuild/DEBIAN/control
	@sed -i "s/ARCHITECTURE/$(DEB_ARCH)/" debbuild/DEBIAN/control
	@sed -i "s/SUMMARY/$(SUMMARY)/" debbuild/DEBIAN/control
	@sed -i "s#URL#$(URL)#" debbuild/DEBIAN/control
	@fakeroot dpkg -b debbuild/ .
.PHONY: deb

rpm: dist
	@rm -rf rpmbuild && mkdir -p rpmbuild
	@rpmbuild -bb --buildroot $(PWD)/rpmbuild/BUILDROOT \
	  --define="%_topdir $(PWD)/rpmbuild" \
	  --define="%version $(VERSION)" \
	  --define="%release $(RELEASE)" \
	  --define="%maintainer $(MAINTAINER)" \
	  --define="%architecture $(RPM_ARCH)" \
	  --define="%summary $(SUMMARY)" \
	  --define="%url $(URL)" \
	  packagers/novm.spec
	@mv rpmbuild/RPMS/$(RPM_ARCH)/*.rpm .
.PHONY: rpm

packages: deb rpm
.PHONY: packages
