export BOSH_CLIENT=admin
export BOSH_GATEWAY_USER=vcap

test: test-unit test-integration

push: test sys-test-local
	git push

pre-commit: test sys-test-local

watch:
	ginkgo watch -r bosh orchestrator ssh artifact instance

test-ci: setup test

test-unit:
	BASIC_AUTH_BOSH_URL=https://lite-bosh.backup-and-restore.cf-app.com \
	BASIC_AUTH_BOSH_CLIENT_SECRET=`lpass show LiteBoshDirector --password` \
	BASIC_AUTH_BOSH_CERT_PATH=~/workspace/bosh-backup-and-restore-meta/certs/lite-bosh.backup-and-restore.cf-app.com.crt \
	UAA_BOSH_URL=https://lite-bosh-uaa.backup-and-restore.cf-app.com \
	UAA_BOSH_CLIENT_SECRET=`lpass show GardenBoshUAADirectorGCP --password` \
	UAA_BOSH_CERT_PATH=~/workspace/bosh-backup-and-restore-meta/certs/lite-bosh-uaa.backup-and-restore.cf-app.com.crt \
	BOSH_GATEWAY_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	ginkgo -r bosh orchestrator ssh artifact instance

test-integration:
	ginkgo -r integration -nodes 4

bin:
	go build -o bbr github.com/pivotal-cf/bosh-backup-and-restore/cmd/bbr

bin-linux:
	GOOS=linux GOARCH=amd64 go build -o bbr github.com/pivotal-cf/bosh-backup-and-restore/cmd/bbr

generate-fakes:
	go generate ./...

generate:
	ls -F | grep / | grep -v vendor | xargs -IN go generate github.com/pivotal-cf/bosh-backup-and-restore/N/...

setup:
	glide install --strip-vendor --strip-vcs
	go get -u github.com/cloudfoundry/bosh-cli
	go get -u github.com/maxbrunsfeld/counterfeiter
	go get -u github.com/onsi/ginkgo/ginkgo

sys-test-local:
	BOSH_URL=https://lite-bosh.backup-and-restore.cf-app.com \
	BOSH_GATEWAY_HOST=lite-bosh.backup-and-restore.cf-app.com \
	BOSH_CLIENT_SECRET=`lpass show LiteBoshDirector --password` \
	BOSH_CERT_PATH=~/workspace/bosh-backup-and-restore-meta/certs/lite-bosh.backup-and-restore.cf-app.com.crt \
	BOSH_GATEWAY_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=`echo $(DEV_ENV)` \
	ginkgo -r -v system

sys-test-local-with-uaa:
	BOSH_URL=https://lite-bosh-uaa.backup-and-restore.cf-app.com \
	BOSH_GATEWAY_HOST=lite-bosh-uaa.backup-and-restore.cf-app.com \
	BOSH_CLIENT_SECRET=`lpass show GardenBoshUAADirectorGCP --password` \
	BOSH_CERT_PATH=~/workspace/bosh-backup-and-restore-meta/certs/lite-bosh-uaa.backup-and-restore.cf-app.com.crt \
	BOSH_GATEWAY_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=`echo $(DEV_ENV)` \
	ginkgo -r -v system

sys-test-local-260:
	BOSH_URL=https://35.187.10.144 \
	BOSH_GATEWAY_HOST=35.187.10.144 \
	BOSH_CLIENT_SECRET=`lpass show Lite260BoshDirector --password` \
	BOSH_CERT_PATH=~/workspace/bosh-backup-and-restore-meta/garden-bosh-260/certs/rootCA.pem \
	BOSH_GATEWAY_KEY=~/workspace/bosh-backup-and-restore-meta/genesis-bosh/bosh.pem \
	TEST_ENV=`echo $(DEV_ENV)` \
	ginkgo -r -v system

sys-test-ci: setup
	TEST_ENV=ci \
	ginkgo -r -v system

upload-test-releases:
	cd fixtures/releases/redis-test-release && bosh -n create release --force && bosh -t $(BOSH_URL) upload release --rebase

release: setup
	mkdir releases
	GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o releases/bbr github.com/pivotal-cf/bosh-backup-and-restore/cmd/bbr
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=${VERSION}" -o releases/bbr-mac github.com/pivotal-cf/bosh-backup-and-restore/cmd/bbr

clean-docker:
	docker ps -q | xargs -IN -P10 docker kill N
	docker ps -a -q | xargs -IN -P10 docker rm N

setup-local-docker:
	eval `docker-machine env`
