package bosh_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BoshManifestReleaseMapper", func() {
	It("parses a manifest and returns a instance group name to job to release mapping", func() {
		var manifest = `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
`
		releaseMapping := NewBoshManifestReleaseMapping(manifest, []string{"red1"})
		Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis"))
	})

	It("parses a manifest with two jobs from the same release correctly", func() {
		var manifest2jobs = `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
- name: red2
  instances: 1
  jobs:
  - name: redis-client
    release: redis
`
		releaseMapping := NewBoshManifestReleaseMapping(manifest2jobs, []string{"red1", "red2"})

		Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis"))
		Expect(releaseMapping.FindReleaseName("red2", "redis-client")).To(Equal("redis"))
	})

	It("parses a manifest with two jobs from the same instance group", func() {
		manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis
- name: red2
  instances: 1
  jobs:
  - name: redis-server
    release: redis
  - name: redis-client
    release: redis
`
		releaseMapping := NewBoshManifestReleaseMapping(manifest, []string{"red1", "red2"})

		Expect(releaseMapping.FindReleaseName("red2", "redis-client")).To(Equal("redis"))
		Expect(releaseMapping.FindReleaseName("red2", "redis-server")).To(Equal("redis"))
	})

	It("parses a manifest with two identically-named jobs from different releases", func() {
		manifest := `---
instance_groups:
- name: red1
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.0
- name: red2
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.5
`

		releaseMapping := NewBoshManifestReleaseMapping(manifest, []string{"red1", "red2"})

		Expect(releaseMapping.FindReleaseName("red1", "redis-server")).To(Equal("redis-2.0"))
		Expect(releaseMapping.FindReleaseName("red2", "redis-server")).To(Equal("redis-2.5"))
	})

	It("ignores jobs and releases from a instance group which has instance count to be zero", func() {
		manifest := `---
instance_groups:
- name: red1
  instances: 0
  jobs:
  - name: redis-server
    release: redis-2.0
- name: red2
  instances: 1
  jobs:
  - name: redis-server
    release: redis-2.5
`
		releaseMapping := NewBoshManifestReleaseMapping(manifest, []string{"red1", "red2"})

		_, err := releaseMapping.FindReleaseName("red1", "redis-server")
		Expect(err).To(MatchError(ContainSubstring("can't find instance group")))

		_, err = releaseMapping.FindReleaseName("red2", "redis-server")
		Expect(err).NotTo(HaveOccurred())
	})
})
