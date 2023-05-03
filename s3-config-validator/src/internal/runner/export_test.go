package runner

import "github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3"

var NewS3ClientImpl = newS3Client

type NewS3Client func(region, endpoint, id, secret string, useIAMProfile bool) (*s3.S3Client, error)

func SetNewS3Client(s3Client NewS3Client) {
	injectableS3Client = s3Client
}
