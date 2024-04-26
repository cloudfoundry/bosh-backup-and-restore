package ratelimiter_test

import (
	"context"
	"time"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/ratelimiter"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ConnectionRateLimiter", func() {

	Describe("RateLimit", func() {
		Context("success", func() {
			It("rate limits", func(ctx context.Context) {
				rateLimiter, err := ratelimiter.NewConnectionRateLimiter(5, "1s")

				Expect(err).To(BeNil())

				completion := make(chan struct{}, 10)

				for i := 0; i < 10; i++ {
					go func() {
						rateLimiter.RateLimit()
						completion <- struct{}{}
					}()
				}

				time.Sleep(10 * time.Millisecond)
				Expect(completion).To(HaveLen(5))

				time.Sleep(1 * time.Second)
				Expect(completion).To(HaveLen(10))

			}, SpecTimeout(2*time.Second))
		})

		Context("failure", func() {
			It("throws and error if rate limit is less than 1", func() {
				_, err := ratelimiter.NewConnectionRateLimiter(0, "1s")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("less than 1"))
			})

			It("throws and error if rate limit is greater than 100", func() {
				_, err := ratelimiter.NewConnectionRateLimiter(101, "1s")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("greater than 100"))
			})

			It("throws and error if duration is less than 1", func() {
				_, err := ratelimiter.NewConnectionRateLimiter(5, "0s")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("cannot be 0"))
			})

			It("throws and error if duration is greater than 3600", func() {
				_, err := ratelimiter.NewConnectionRateLimiter(5, "3601s")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("greater than 3600"))
			})

			It("throws and error if duration is invalid", func() {
				_, err := ratelimiter.NewConnectionRateLimiter(5, "1yxz")

				Expect(err).NotTo(BeNil())
				Expect(err.Error()).To(ContainSubstring("duration \"1yxz\""))
			})
		})
	})
})
