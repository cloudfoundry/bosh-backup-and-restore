export BOSH_CLIENT=admin

test: test-unit test-integration

push: test sys-test-local
	git push

pre-commit: test sys-test-local

watch:
	ginkgo watch -r -skipPackage integration,system,backup

test-ci: setup test

test-unit:
	ginkgo -p -r -skipPackage integration,system

test-integration:
	ginkgo -r -trace integration

bin:
	go build -o bbr github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr

bin-linux:
	GOOS=linux GOARCH=amd64 go build -o bbr github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr

generate-fakes:
	go generate ./...

generate:
	ls -F | grep / | grep -v vendor | xargs -IN go generate github.com/cloudfoundry-incubator/bosh-backup-and-restore/N/...

setup:
	dep ensure
	go get -u github.com/cloudfoundry/bosh-cli
	go get -u github.com/maxbrunsfeld/counterfeiter
	go get -u github.com/onsi/ginkgo/ginkgo

sys-test-local: sys-test-local-deployment sys-test-local-director

sys-test-local-deployment:
	BOSH_ENVIRONMENT=https://lite-bosh.backup-and-restore.cf-app.com \
	BOSH_GW_HOST=lite-bosh.backup-and-restore.cf-app.com \
	BOSH_CLIENT_SECRET=`lpass show LiteBoshDirector --password` \
	BOSH_CA_CERT=~/workspace/bosh-backup-and-restore-meta/certs/lite-bosh.backup-and-restore.cf-app.com.crt \
	BOSH_GW_USER=vcap \
	BOSH_GW_PRIVATE_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=dev \
	ginkgo -r -v -trace system/deployment

sys-test-local-director:
	BOSH_ENVIRONMENT=https://genesis-bosh.backup-and-restore.cf-app.com \
	BOSH_GW_HOST=genesis-bosh.backup-and-restore.cf-app.com \
	BOSH_CLIENT_SECRET=`lpass show GenesisBoshDirectorGCP --password` \
	BOSH_CA_CERT=~/workspace/bosh-backup-and-restore-meta/certs/genesis-bosh.backup-and-restore.cf-app.com.crt \
	BOSH_GW_PRIVATE_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	BOSH_GW_USER=vcap \
	SSH_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	HOST_TO_BACKUP=10.0.0.8 \
	TEST_ENV=ci \
	ginkgo -r -v -trace system/director

sys-test-director-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/director

sys-test-deployment-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/deployment

sys-test-windows-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/windows

sys-test-bosh-all-proxy-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/bosh_all_proxy

sys-test-local-with-uaa:
	BOSH_ENVIRONMENT=https://lite-bosh-uaa.backup-and-restore.cf-app.com \
	BOSH_GW_HOST=lite-bosh-uaa.backup-and-restore.cf-app.com \
	BOSH_CLIENT_SECRET=`lpass show GardenBoshUAADirectorGCP --password` \
	BOSH_CA_CERT=~/workspace/bosh-backup-and-restore-meta/certs/lite-bosh-uaa.backup-and-restore.cf-app.com.crt \
    BOSH_GW_USER=jumpbox \
	BOSH_GW_PRIVATE_KEY=~/workspace/bosh-backup-and-restore-meta/garden-bosh-uaa/bosh.pem \
	TEST_ENV=dev \
	ginkgo -r -v -trace system/deployment

sys-test-local-260:
	BOSH_ENVIRONMENT=https://35.187.10.144 \
	BOSH_GW_HOST=35.187.10.144 \
	BOSH_CLIENT_SECRET=`lpass show Lite260BoshDirector --password` \
	BOSH_CA_CERT=~/workspace/bosh-backup-and-restore-meta/garden-bosh-260/certs/rootCA.pem \
	BOSH_GW_PRIVATE_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=dev \
	ginkgo -r -v -trace system/deployment


upload-test-releases:
	fixtures/releases/upload-release redis-test-release && \
	fixtures/releases/upload-release many-bbr-jobs-release

release: setup
	go version
	mkdir releases
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o releases/bbr github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o releases/bbr-mac github.com/cloudfoundry-incubator/bosh-backup-and-restore/cmd/bbr
	cd releases && shasum -a 256 * > checksum.sha256

clean-docker:
	docker ps -q | xargs -IN -P10 docker kill N
	docker ps -a -q | xargs -IN -P10 docker rm N

setup-local-docker:
	eval `docker-machine env`
