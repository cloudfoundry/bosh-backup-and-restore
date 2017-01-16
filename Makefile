export BOSH_CLIENT=admin
export BOSH_URL=https://lite-bosh.backup-and-restore.cf-app.com
export BOSH_GATEWAY_USER=vcap
export BOSH_GATEWAY_HOST=lite-bosh.backup-and-restore.cf-app.com

test: test-unit test-integration

push: test sys-test-local
	git push

pre-commit: test sys-test-local

watch:
	ginkgo watch -r boshclient backuper integration

test-ci: setup test

test-unit:
	ginkgo -v ssh
	ginkgo -r bosh backuper ssh artifact

test-integration:
	ginkgo -r integration -nodes 4

bin:
	go build -o pbr github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr

bin-linux:
	GOOS=linux GOARCH=amd64 go build -o pbr github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr

generate-fakes:
	go generate ./...

generate:
	ls -F | grep / | grep -v vendor | xargs -IN go generate github.com/pivotal-cf/pcf-backup-and-restore/N/...

setup:
	glide install --strip-vendor --strip-vcs
	go get github.com/cloudfoundry/bosh-cli
	go get github.com/maxbrunsfeld/counterfeiter
	go get github.com/onsi/ginkgo/ginkgo

sys-test-local:
	BOSH_CLIENT_SECRET=`lpass show LiteBoshDirector --password` \
	BOSH_CERT_PATH=~/workspace/pcf-backup-and-restore-meta/certs/lite-bosh.backup-and-restore.cf-app.com.crt \
	BOSH_GATEWAY_KEY=~/workspace/pcf-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=`echo $(DEV_ENV)` \
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

clean-docker:
	docker ps -q | xargs -IN -P10 docker kill N
	docker ps -a -q | xargs -IN -P10 docker rm N
