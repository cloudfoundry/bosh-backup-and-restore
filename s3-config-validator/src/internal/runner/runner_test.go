package runner_test

import (
	"errors"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/config"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3"
	"io"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/probe"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/runner"
)

func SucceedingProbe(string) error {
	return nil
}

func FailingProbe(string) error {
	return errors.New("FailingProbe")
}

var _ = Describe("ProbeRunner", func() {
	var probeRunner ProbeRunner
	var writer io.Writer

	When("all probes succeed", func() {

		BeforeEach(func() {
			probeSet := []probe.NamedProbe{
				{Name: "Probe one", Probe: SucceedingProbe},
				{Name: "Probe two", Probe: SucceedingProbe},
			}

			writer = gbytes.NewBuffer()

			bucket := Bucket{
				Resource: "test-resource",
				Name:     "test-bucket",
				Type:     Live,
			}

			probeRunner = ProbeRunner{
				Bucket:   bucket,
				ProbeSet: probeSet,
				Writer:   writer,
			}
		})

		It("returns true", func() {
			result := probeRunner.Run()

			Expect(result).To(BeTrue())
		})

		It("writes the probe results", func() {
			probeRunner.Run()

			Eventually(writer).Should(gbytes.Say("Validating test-resource's live bucket test-bucket ..."))
			Eventually(writer).Should(gbytes.Say(" * Probe one ... Yes"))
			Eventually(writer).Should(gbytes.Say(" * Probe two ... Yes"))
		})
	})

	When("some probes fail", func() {
		BeforeEach(func() {
			probeSet := []probe.NamedProbe{
				{Name: "Probe one", Probe: FailingProbe},
				{Name: "Probe two", Probe: SucceedingProbe},
			}

			writer = gbytes.NewBuffer()

			bucket := Bucket{
				Resource: "test-resource",
				Name:     "test-bucket",
				Type:     Live,
			}

			probeRunner = ProbeRunner{
				Bucket:   bucket,
				ProbeSet: probeSet,
				Writer:   writer,
			}
		})
		It("returns false", func() {
			result := probeRunner.Run()

			Expect(result).To(BeFalse())
		})

		It("displays the probe results", func() {
			probeRunner.Run()

			Eventually(writer).Should(gbytes.Say("Validating test-resource's live bucket test-bucket ..."))
			Eventually(writer).Should(gbytes.Say(` * Probe one ... No \[reason: FailingProbe\]`))
			Eventually(writer).Should(gbytes.Say(" * Probe two ... Yes"))
		})
	})
})

type NewS3ClientArgs struct {
	Region, Endpoint, Id, Secret string
}

var _ = Describe("Versioned", func() {

	It("should construct a live probe runner", func() {
		var newS3ClientArgs []NewS3ClientArgs

		SetNewS3Client(func(region, endpoint, id, secret string) (*s3.S3Client, error) {
			newS3ClientArgs = append(newS3ClientArgs, NewS3ClientArgs{
				Region:   region,
				Endpoint: endpoint,
				Id:       id,
				Secret:   secret,
			})
			return NewS3ClientImpl(region, endpoint, id, secret)
		})

		probeRunners := NewProbeRunners(
			"test-versioned-resource",
			config.LiveBucket{
				Region:   "test-live-region",
				Endpoint: "test-live-endpoint",
				ID:       "test-id",
				Secret:   "test-secret",
				Name:     "test-live-bucket",
				Backup: &config.BackupBucket{},
			},
			true,
			true,
		)

		Expect(newS3ClientArgs).To(ConsistOf(
			NewS3ClientArgs{
				Region:   "test-live-region",
				Endpoint: "test-live-endpoint",
				Id:       "test-id",
				Secret:   "test-secret",
			}))

		var buckets []Bucket
		for _, probeRunner := range probeRunners {
			buckets = append(buckets, probeRunner.Bucket)
		}
		Expect(buckets).To(ConsistOf(
			Bucket{
				Resource: "test-versioned-resource",
				Name:     "test-live-bucket",
				Type:     Live,
			}))
	})
})

var _ = Describe("Unversioned", func() {

	It("should construct a live and a backup bucket probe runner", func() {
		var newS3ClientArgs []NewS3ClientArgs

		SetNewS3Client(func(region, endpoint, id, secret string) (*s3.S3Client, error) {
			newS3ClientArgs = append(newS3ClientArgs, NewS3ClientArgs{
				Region:   region,
				Endpoint: endpoint,
				Id:       id,
				Secret:   secret,
			})
			return NewS3ClientImpl(region, endpoint, id, secret)
		})

		probeRunners := NewProbeRunners(
			"test-unversioned-resource",
			config.LiveBucket{
				Region:   "test-live-region",
				Endpoint: "test-live-endpoint",
				ID:       "test-id",
				Secret:   "test-secret",
				Name:     "test-live-bucket",
				Backup: &config.BackupBucket{
					Name:   "test-backup-bucket",
					Region: "test-backup-region",
				},
			},
			true,
			false,
		)

		Expect(newS3ClientArgs).To(ConsistOf(
			NewS3ClientArgs{
				Region:   "test-live-region",
				Endpoint: "test-live-endpoint",
				Id:       "test-id",
				Secret:   "test-secret",
			},
			NewS3ClientArgs{
				Region:   "test-backup-region",
				Endpoint: "test-live-endpoint",
				Id:       "test-id",
				Secret:   "test-secret",
			}))

		var buckets []Bucket
		for _, probeRunner := range probeRunners {
			buckets = append(buckets, probeRunner.Bucket)
		}
		Expect(buckets).To(ConsistOf(
			Bucket{
				Resource: "test-unversioned-resource",
				Name:     "test-live-bucket",
				Type:     Live,
			},
			Bucket{
				Resource: "test-unversioned-resource",
				Name:     "test-backup-bucket",
				Type:     Backup,
			}))
	})
})
