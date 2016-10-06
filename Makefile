test: test-unit test-integration

test-unit:
	ginkgo -r boshclient backuper

test-integration:
	ginkgo -r integration

sys-test-local:
	BOSH_PASSWORD=admin BOSH_USERNAME=admin BOSH_URL="https://52.50.223.208:25555" ginkgo -r system

bin:
	go build -o pbr  github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr

generate-fakes:
	go generate ./...

setup:
	glide install
