# Makefile used to create packages for clr-dissector. It doesn't assume
# that the code is inside a GOPATH, and always copy the files into a
# new workspace to get the work done. Go tools doesn't reliably work
# with symbolic links.
#
# For historical purposes, it also works in a development environment
# when the repository is already inside a GOPATH.

.NOTPARALLEL:

GO_PACKAGE_PREFIX := github.intel.com/crlynch/clr-dissector

.PHONY: gopath

VERSION=0.0.1
# Strictly speaking we should check if it the directory is inside an
# actual GOPATH, but the directory structure matching is likely enough.
ifeq (,$(findstring ${GO_PACKAGE_PREFIX},${CURDIR}))
LOCAL_GOPATH := ${CURDIR}/.gopath
export GOPATH := ${LOCAL_GOPATH}
gopath:
	@rm -rf ${LOCAL_GOPATH}/src
	@mkdir -p ${LOCAL_GOPATH}/src/${GO_PACKAGE_PREFIX}
	@cp -af * ${LOCAL_GOPATH}/src/${GO_PACKAGE_PREFIX}
	@echo "Prepared a local GOPATH=${GOPATH}"
	@echo ${GO_PACKAGE_PREFIX}
else
LOCAL_GOPATH :=
GOPATH ?= ${HOME}/go
gopath:
	@echo "Code already in existing GOPATH=${GOPATH}"
endif

.PHONY: build install clean check

.DEFAULT_GOAL := build

build: gopath
	go install ${GO_PACKAGE_PREFIX}/cmd/bundles2packages
	go install ${GO_PACKAGE_PREFIX}/cmd/dissector
	go install ${GO_PACKAGE_PREFIX}/cmd/downloadpackages
	go install ${GO_PACKAGE_PREFIX}/cmd/downloadrepo
	go install ${GO_PACKAGE_PREFIX}/cmd/image2bundles
	go install ${GO_PACKAGE_PREFIX}/cmd/packages2packages
	go install ${GO_PACKAGE_PREFIX}/cmd/packages2source
	go install ${GO_PACKAGE_PREFIX}/cmd/packages2files

install: gopath
	test -d $(DESTDIR)/usr/bin || install -D -d -m 00755 $(DESTDIR)/usr/bin;
	install -m 00755 $(GOPATH)/bin/bundles2packages $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/dissector $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/downloadpackages $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/downloadrepo $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/image2bundles $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/packages2packages $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/packages2source $(DESTDIR)/usr/bin/.
	install -m 00755 $(GOPATH)/bin/packages2files $(DESTDIR)/usr/bin/.

check: gopath
	go test -cover ${GO_PACKAGE_PREFIX}/...


.PHONY: lint
lint: gopath
	@gometalinter.v2 --deadline=10m --tests --vendor --disable-all \
	--enable=misspell \
	--enable=vet \
	--enable=ineffassign \
	--enable=gofmt \
	$${CYCLO_MAX:+--enable=gocyclo --cyclo-over=$${CYCLO_MAX}} \
	--enable=golint \
	--enable=deadcode \
	--enable=varcheck \
	--enable=structcheck \
	--enable=unused \
	--enable=vetshadow \
	--enable=errcheck \
	./...

clean:
ifeq (,${LOCAL_GOPATH})
	go clean -i -x
else
	rm -rf ${LOCAL_GOPATH}
endif
	rm -f clr-dissecor-*.tar.gz

release:
	@if [ ! -d .git ]; then \
		echo "Release needs to be used from a git repository"; \
		exit 1; \
	fi
	git archive --format=tar.gz --verbose -o clr-dissector-${VERSION}.tar.gz HEAD --prefix=clr-dissector-${VERSION}/
