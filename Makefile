

sys-test-local: 
	cd system
	BOSH_PASSWORD=admin BOSH_USERNAME=admin BOSH_URL="https://52.50.223.208:25555" ginkgo -r

make-bin:
	go build -o  github.com/pivotal-cf/pcf-backup-and-restore/cmd/pbr
