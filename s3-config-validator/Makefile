.PHONY: artifact build test-e2e test-unit

SHELL=/usr/bin/env bash -o pipefail

all: test-unit test-e2e artifact

artifact: build
	cd build && \
	cp ../README.md . && \
	tar cf artifact.tgz *

build:
	./build_all.sh

ifeq ($(shell whoami), root)
NON_ROOT_USER='bbr'
test-unit:
	cd src/ \
	&& chmod -R g+w . \
	&& useradd --groups root --create-home $(NON_ROOT_USER) \
	&& set +x; \
		su $(NON_ROOT_USER) -c 'go run github.com/onsi/ginkgo/v2/ginkgo -r --keep-going internal'; \
		status=$$?; \
		userdel --remove bbr \
	&& exit $${status}
else
test-unit:
	cd src/ \
 	&& go run github.com/onsi/ginkgo/v2/ginkgo -r --keep-going internal
endif

test-e2e:
	cd src/test && \
	go run github.com/onsi/ginkgo/v2/ginkgo -r --keep-going . \
	| sed 's/"\(aws_.*\)"\: "\(.*\)"/"\1": "<redacted>"/g'
