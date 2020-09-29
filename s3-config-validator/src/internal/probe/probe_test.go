package probe_test

import (
	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/probe"
	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3/s3fakes"
)

type ProbeResult struct {
	Name      string
	Succeeded bool
}

var _ = Describe("ProbeSet", func() {
	var (
		bucket       = "test-bucket"
		probeSet     Set
		fakeS3Client s3fakes.FakeClient
	)

	BeforeEach(func() {
		fakeS3Client = s3fakes.FakeClient{}
	})

	Describe("when running probes", func() {

		Context("versioned", func() {
			Context("read-only", func() {
				BeforeEach(func() {
					fakeS3Client.IsVersionedReturns(nil)
					fakeS3Client.CanListObjectVersionsReturns(nil)
					fakeS3Client.CanGetObjectVersionsReturns(nil)

					probeSet = NewSet(&fakeS3Client, true, true)
				})

				It("successfully runs just read-only probes", func() {
					Expect(runAllProbesAgainstBucket(probeSet, bucket)).To(Equal([]ProbeResult{
						{
							Name:      "Bucket is versioned",
							Succeeded: true,
						},
						{
							Name:      "Can list object versions",
							Succeeded: true,
						},
						{
							Name:      "Can get object versions",
							Succeeded: true,
						},
					}))
					Expect(fakeS3Client.IsVersionedCallCount()).To(Equal(1))
					Expect(fakeS3Client.IsVersionedArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanListObjectVersionsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanListObjectVersionsArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanGetObjectVersionsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanGetObjectVersionsArgsForCall(0)).To(Equal(bucket))


					Expect(fakeS3Client.CanPutObjectsCallCount()).To(BeZero())
					Expect(fakeS3Client.CanListObjectsCallCount()).To(BeZero())
					Expect(fakeS3Client.CanGetObjectsCallCount()).To(BeZero())
					Expect(fakeS3Client.IsUnversionedCallCount()).To(BeZero())

				})
			})

			Context("read-write", func() {
				BeforeEach(func() {
					fakeS3Client.IsVersionedReturns(nil)
					fakeS3Client.CanListObjectVersionsReturns(nil)
					fakeS3Client.CanGetObjectVersionsReturns(nil)
					fakeS3Client.CanPutObjectsReturns(nil)

					probeSet = NewSet(&fakeS3Client, false, true)
				})

				It("successfully runs all probes", func() {
					Expect(runAllProbesAgainstBucket(probeSet, bucket)).To(Equal([]ProbeResult{
						{
							Name:      "Bucket is versioned",
							Succeeded: true,
						},
						{
							Name:      "Can list object versions",
							Succeeded: true,
						},
						{
							Name:      "Can get object versions",
							Succeeded: true,
						},
						{
							Name:      "Can put objects",
							Succeeded: true,
						},
					}))
					Expect(fakeS3Client.IsVersionedCallCount()).To(Equal(1))
					Expect(fakeS3Client.IsVersionedArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanListObjectVersionsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanListObjectVersionsArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanGetObjectVersionsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanGetObjectVersionsArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanPutObjectsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanPutObjectsArgsForCall(0)).To(Equal(bucket))

					Expect(fakeS3Client.CanListObjectsCallCount()).To(BeZero())
					Expect(fakeS3Client.CanGetObjectsCallCount()).To(BeZero())
					Expect(fakeS3Client.IsUnversionedCallCount()).To(BeZero())
				})
			})

			Context("failing validation", func() {
				When("All probes fail", func() {
					BeforeEach(func() {
						fakeS3Client.IsVersionedReturns(errors.New("bucket is versioned error"))
						fakeS3Client.CanListObjectVersionsReturns(errors.New("bucket is versioned error"))
						fakeS3Client.CanGetObjectVersionsReturns(errors.New("bucket is versioned error"))
						fakeS3Client.CanPutObjectsReturns(errors.New("bucket is versioned error"))

						probeSet = NewSet(&fakeS3Client, false, true)
					})

					It("returns all errors", func() {
						Expect(runAllProbesAgainstBucket(probeSet, bucket)).To(ConsistOf(
							ProbeResult{
								Name:      "Bucket is versioned",
								Succeeded: false,
							},
							ProbeResult{
								Name:      "Can list object versions",
								Succeeded: false,
							},
							ProbeResult{
								Name:      "Can get object versions",
								Succeeded: false,
							},
							ProbeResult{
								Name:      "Can put objects",
								Succeeded: false,
							},
						))
						Expect(fakeS3Client.IsVersionedCallCount()).To(Equal(1))
						Expect(fakeS3Client.IsVersionedArgsForCall(0)).To(Equal(bucket))
						Expect(fakeS3Client.CanListObjectVersionsCallCount()).To(Equal(1))
						Expect(fakeS3Client.CanListObjectVersionsArgsForCall(0)).To(Equal(bucket))
						Expect(fakeS3Client.CanGetObjectVersionsCallCount()).To(Equal(1))
						Expect(fakeS3Client.CanGetObjectVersionsArgsForCall(0)).To(Equal(bucket))
						Expect(fakeS3Client.CanPutObjectsCallCount()).To(Equal(1))
						Expect(fakeS3Client.CanPutObjectsArgsForCall(0)).To(Equal(bucket))

						Expect(fakeS3Client.IsUnversionedCallCount()).To(BeZero())
						Expect(fakeS3Client.CanGetObjectsCallCount()).To(BeZero())
						Expect(fakeS3Client.CanListObjectsCallCount()).To(BeZero())
					})
				})
			})
		})

		Context("unversioned", func() {
			Context("read-only", func() {
				BeforeEach(func() {
					fakeS3Client.IsUnversionedReturns(nil)
					fakeS3Client.CanListObjectsReturns(nil)
					fakeS3Client.CanGetObjectsReturns(nil)

					probeSet = NewSet(&fakeS3Client, true, false)
				})

				It("successfully runs just read-only probes", func() {
					Expect(runAllProbesAgainstBucket(probeSet, bucket)).To(Equal([]ProbeResult{
						{
							Name:      "Bucket is not versioned",
							Succeeded: true,
						},
						{
							Name:      "Can list objects",
							Succeeded: true,
						},
						{
							Name:      "Can get objects",
							Succeeded: true,
						},
					}))
					Expect(fakeS3Client.IsUnversionedCallCount()).To(Equal(1))
					Expect(fakeS3Client.IsUnversionedArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanListObjectsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanListObjectsArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanGetObjectsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanGetObjectsArgsForCall(0)).To(Equal(bucket))

					Expect(fakeS3Client.CanPutObjectsCallCount()).To(BeZero())
					Expect(fakeS3Client.IsVersionedCallCount()).To(BeZero())
					Expect(fakeS3Client.CanListObjectVersionsCallCount()).To(BeZero())
				})
			})

			Context("read-write", func() {
				BeforeEach(func() {
					fakeS3Client.IsUnversionedReturns(nil)
					fakeS3Client.CanListObjectsReturns(nil)
					fakeS3Client.CanGetObjectsReturns(nil)
					fakeS3Client.CanPutObjectsReturns(nil)

					probeSet = NewSet(&fakeS3Client, false, false)
				})

				It("successfully runs all probes", func() {
					Expect(runAllProbesAgainstBucket(probeSet, bucket)).To(Equal([]ProbeResult{
						{
							Name:      "Bucket is not versioned",
							Succeeded: true,
						},
						{
							Name:      "Can list objects",
							Succeeded: true,
						},
						{
							Name:      "Can get objects",
							Succeeded: true,
						},
						{
							Name:      "Can put objects",
							Succeeded: true,
						},
					}))
					Expect(fakeS3Client.IsUnversionedCallCount()).To(Equal(1))
					Expect(fakeS3Client.IsUnversionedArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanListObjectsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanListObjectsArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanGetObjectsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanGetObjectsArgsForCall(0)).To(Equal(bucket))
					Expect(fakeS3Client.CanPutObjectsCallCount()).To(Equal(1))
					Expect(fakeS3Client.CanPutObjectsArgsForCall(0)).To(Equal(bucket))

					Expect(fakeS3Client.IsVersionedCallCount()).To(BeZero())
					Expect(fakeS3Client.CanListObjectVersionsCallCount()).To(BeZero())
				})
			})

			Context("failing validation", func() {
				When("All probes fail", func() {
					BeforeEach(func() {
						fakeS3Client.IsUnversionedReturns(errors.New("bucket is versioned error"))
						fakeS3Client.CanListObjectsReturns(errors.New("bucket is versioned error"))
						fakeS3Client.CanGetObjectsReturns(errors.New("bucket is versioned error"))
						fakeS3Client.CanPutObjectsReturns(errors.New("bucket is versioned error"))

						probeSet = NewSet(&fakeS3Client, false, false)
					})

					It("returns all errors", func() {
						Expect(runAllProbesAgainstBucket(probeSet, bucket)).To(ConsistOf(
							ProbeResult{
								Name:      "Bucket is not versioned",
								Succeeded: false,
							},
							ProbeResult{
								Name:      "Can list objects",
								Succeeded: false,
							},
							ProbeResult{
								Name:      "Can get objects",
								Succeeded: false,
							},
							ProbeResult{
								Name:      "Can put objects",
								Succeeded: false,
							},
						))
						Expect(fakeS3Client.IsUnversionedCallCount()).To(Equal(1))
						Expect(fakeS3Client.IsUnversionedArgsForCall(0)).To(Equal(bucket))
						Expect(fakeS3Client.CanListObjectsCallCount()).To(Equal(1))
						Expect(fakeS3Client.CanListObjectsArgsForCall(0)).To(Equal(bucket))
						Expect(fakeS3Client.CanGetObjectsCallCount()).To(Equal(1))
						Expect(fakeS3Client.CanGetObjectsArgsForCall(0)).To(Equal(bucket))
						Expect(fakeS3Client.CanPutObjectsCallCount()).To(Equal(1))
						Expect(fakeS3Client.CanPutObjectsArgsForCall(0)).To(Equal(bucket))

						Expect(fakeS3Client.IsVersionedCallCount()).To(BeZero())
						Expect(fakeS3Client.CanListObjectVersionsCallCount()).To(BeZero())
					})
				})
			})
		})

	})
})

func runAllProbesAgainstBucket(probeSet Set, bucket string) (probeResults []ProbeResult) {
	for _, probe := range probeSet {
		probeResults = append(
			probeResults,
			ProbeResult{
				Name:      probe.Name,
				Succeeded: probe.Probe(bucket) == nil,
			},
		)
	}

	return
}
