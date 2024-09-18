package ratelimiter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRatelimiter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Ratelimiter Suite")
}
