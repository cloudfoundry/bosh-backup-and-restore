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

sys-test-director-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/director

sys-test-deployment-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/deployment

sys-test-windows-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/windows

sys-test-all-deployments-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/all_deployments

sys-test-bosh-all-proxy-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v -trace system/bosh_all_proxy

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
