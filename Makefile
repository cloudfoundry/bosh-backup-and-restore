export BOSH_PASSWORD=admin
export BOSH_USER=admin
export BOSH_URL=https://52.50.223.208:25555

test: test-unit test-integration

test-ci: setup test

test-unit:
	ginkgo -r boshclient backuper

test-integration:
	ginkgo -r integration

bin:
	go build -o pbr github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr

generate-fakes:
	go generate ./...

setup:
	glide install

sys-test-local: setup-sys-test-local
	BOSH_TEST_DEPLOYMENT=systest-dev ginkgo -r system

setup-sys-test-local:
	bosh -t $(BOSH_URL) -n -d fixtures/systest-dev.yml deploy

sys-test-ci: setup-sys-test-ci
	BOSH_TEST_DEPLOYMENT=systest-ci ginkgo -r system

setup-sys-test-ci: setup
	bosh -t $(BOSH_URL) -n -d fixtures/systest-ci.yml deploy

dev_version := $(shell git rev-parse HEAD | cut -c1-6 | tr -d '\n')
release: setup
	mkdir -p releases/release-${dev_version}
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=0.0.0-${dev_version}" -o releases/release-${dev_version}/pbr github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=0.0.0-${dev_version}" -o releases/release-$(dev_version)/pbr-mac github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr
