.PHONY: build test-unit test-e2e artifact

SHELL=/usr/bin/env bash -o pipefail

all: test-unit test-e2e artifact

artifact: build
	cd build && \
	cp ../README.md . && \
	tar cf artifact.tgz *

build:
	./build_all.sh

test-unit:
	cd src/ && \
	go run github.com/onsi/ginkgo/v2/ginkgo -r --keep-going internal

test-e2e:
	cd src/test && \
	go run github.com/onsi/ginkgo/v2/ginkgo -r --keep-going . \
	| sed 's/"\(aws_.*\)"\: "\(.*\)"/"\1": "<redacted>"/g'
