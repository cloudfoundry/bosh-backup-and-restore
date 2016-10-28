export BOSH_PASSWORD=admin
export BOSH_USER=admin
export BOSH_URL=https://lite-bosh.backup-and-restore.cf-app.com
export BOSH_GATEWAY_USER=vcap
export BOSH_GATEWAY_HOST=lite-bosh.backup-and-restore.cf-app.com

test: test-unit test-integration

pre-commit: test sys-test-local

watch:
	ginkgo watch -r boshclient backuper integration

test-ci: setup test

test-unit:
	ginkgo -r bosh backuper ssh

test-integration:
	ginkgo -r integration

bin:
	go build -o pbr github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr

generate-fakes:
	go generate ./...

setup:
	glide install --strip-vendor --strip-vcs

sys-test-local:
	BOSH_CERT_PATH=~/workspace/pcf-backup-and-restore-meta/certs/lite-bosh.backup-and-restore.cf-app.com.crt \
	BOSH_GATEWAY_KEY=~/workspace/pcf-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=dev \
	ginkgo -r -v system

sys-test-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v system

upload-test-releases:
	cd fixtures/releases/redis-test-release && bosh -n create release --force && bosh -t $(BOSH_URL) upload release --rebase

release: setup
	mkdir releases
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o releases/pbr github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o releases/pbr-mac github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr
