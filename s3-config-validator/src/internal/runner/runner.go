package runner

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/config"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/probe"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3"
)

type Runner interface {
	Run() bool
}

type ProbeRunner struct {
	Bucket   Bucket
	ProbeSet probe.Set
	Writer   io.Writer
}

type BucketType string

const (
	Live   BucketType = "live"
	Backup BucketType = "backup"
)

type Bucket struct {
	Resource string
	Name     string
	Type     BucketType
}

func (b Bucket) String() string {
	if strings.HasSuffix(b.Resource, "s") {
		return fmt.Sprintf("%s' %s bucket %s", b.Resource, b.Type, b.Name)
	}

	return fmt.Sprintf("%s's %s bucket %s", b.Resource, b.Type, b.Name)
}

func (r *ProbeRunner) Run() (succeeded bool) {
	succeeded = true

	fmt.Fprintf(r.Writer, "Validating %s ...\n", r.Bucket)

	for _, probe := range r.ProbeSet {
		fmt.Fprintf(r.Writer, " * %s ... ", probe.Name)

		err := probe.Probe(r.Bucket.Name)

		if err != nil {
			succeeded = false

			fmt.Fprintf(r.Writer, "No [reason: %s]\n", err.Error())
		} else {
			fmt.Fprint(r.Writer, "Yes\n")
		}
	}

	fmt.Fprintf(r.Writer, "\n")

	return
}

func NewProbeRunners(resource string, bucket config.LiveBucket, readOnly, versioned bool) []ProbeRunner {
	// the s3 clients for the runners are meant to be constructed the same way as the BBR SDK is constructing them;
	// see https://github.com/cloudfoundry-incubator/backup-and-restore-sdk-release/blob/59d6a95963d0a81e77b666f44338833c45452d37/src/s3-blobstore-backup-restore/unversioned/config.go#L32-L61
	// while the respective regions are being used, the endpoint is the same for both.
	// this is a known issue: https://www.pivotaltracker.com/story/show/174547239

	liveProbeRunner := NewProbeRunner(
		bucket.Region, bucket.Endpoint, bucket.ID, bucket.Secret,
		Bucket{
			Resource: resource,
			Name:     bucket.Name,
			Type:     Live,
		},
		readOnly,
		versioned,
		bucket.UseIAMProfile,
	)

	if !versioned {
		backupProbeRunner := NewProbeRunner(
			bucket.Backup.Region, bucket.Endpoint, bucket.ID, bucket.Secret,
			Bucket{
				Resource: resource,
				Name:     bucket.Backup.Name,
				Type:     Backup,
			},
			readOnly,
			false,
			bucket.UseIAMProfile,
		)
		return []ProbeRunner{liveProbeRunner, backupProbeRunner}
	}

	return []ProbeRunner{liveProbeRunner}

}

var injectableS3Client = newS3Client

func NewProbeRunner(region, endpoint, id, secret string, bucket Bucket, readOnly, versioned, useIAMProfile bool) ProbeRunner {
	s3Client, _ := injectableS3Client(region, endpoint, id, secret, useIAMProfile)

	probeSet := probe.NewSet(s3Client, readOnly, versioned)

	return ProbeRunner{
		ProbeSet: probeSet,
		Writer:   os.Stdout,
		Bucket:   bucket,
	}
}

func newS3Client(region, endpoint, id, secret string, useIAMProfile bool) (*s3.S3Client, error) {
	return s3.NewS3Client(region, endpoint, id, secret, false)
}
