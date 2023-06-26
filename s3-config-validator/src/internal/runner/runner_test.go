package runner_test

import (
	"errors"
	"io"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/config"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/probe"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/runner"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3"
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
	UseIAMProfile                bool
}

var _ = Describe("Versioned", func() {
	When("we are not using IAM Profiles", func() {
		It("should construct a live probe runner with bucket secrets", func() {
			var newS3ClientArgs []NewS3ClientArgs

			SetNewS3Client(func(region, endpoint, id, secret, role string, useIAMProfile bool) (*s3.S3Client, error) {
				newS3ClientArgs = append(newS3ClientArgs, NewS3ClientArgs{
					Region:        region,
					Endpoint:      endpoint,
					Id:            id,
					Secret:        secret,
					UseIAMProfile: useIAMProfile,
				})
				return NewS3ClientImpl(region, endpoint, id, secret, role, useIAMProfile)
			})

			probeRunners := NewProbeRunners(
				"test-versioned-resource",
				config.LiveBucket{
					Region:        "test-live-region",
					Endpoint:      "test-live-endpoint",
					ID:            "test-id",
					Secret:        "test-secret",
					Name:          "test-live-bucket",
					Backup:        &config.BackupBucket{},
					UseIAMProfile: false,
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

	When("we are using IAM Profiles", func() {
		It("should construct a live probe runner without bucket secrets", func() {
			var newS3ClientArgs []NewS3ClientArgs

			SetNewS3Client(func(region, endpoint, id, secret, role string, useIAMProfile bool) (*s3.S3Client, error) {
				newS3ClientArgs = append(newS3ClientArgs, NewS3ClientArgs{
					Region:        region,
					Endpoint:      endpoint,
					Id:            id,
					Secret:        secret,
					UseIAMProfile: useIAMProfile,
				})
				return NewS3ClientImpl(region, endpoint, id, secret, role, useIAMProfile)
			})

			NewProbeRunners(
				"test-versioned-resource",
				config.LiveBucket{
					Region:        "test-live-region",
					Endpoint:      "test-live-endpoint",
					ID:            "",
					Secret:        "",
					Name:          "test-live-bucket",
					Backup:        &config.BackupBucket{},
					UseIAMProfile: true,
				},
				true,
				true,
			)

			Expect(newS3ClientArgs).To(ConsistOf(
				NewS3ClientArgs{
					Region:        "test-live-region",
					Endpoint:      "test-live-endpoint",
					Id:            "",
					Secret:        "",
					UseIAMProfile: true,
				}))
		})
	})
})

var _ = Describe("Unversioned", func() {

	It("should construct a live and a backup bucket probe runner", func() {
		var newS3ClientArgs []NewS3ClientArgs

		SetNewS3Client(func(region, endpoint, id, secret, role string, useIAMProfile bool) (*s3.S3Client, error) {
			newS3ClientArgs = append(newS3ClientArgs, NewS3ClientArgs{
				Region:        region,
				Endpoint:      endpoint,
				Id:            id,
				Secret:        secret,
				UseIAMProfile: useIAMProfile,
			})
			return NewS3ClientImpl(region, endpoint, id, secret, role, useIAMProfile)
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
