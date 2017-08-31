package bosh_test

import (
	. "github.com/cloudfoundry-incubator/bosh-backup-and-restore/bosh"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ReleaseMapper", func() {

	var exampleManifest = `---
instance_groups:
- name: red1
  jobs:
  - name: redis-server
    release: redis
`
	var exampleManifest2jobs = `---
instance_groups:
- name: red1
  jobs:
  - name: redis-server
    release: redis
- name: red2
  jobs:
  - name: redis-client
    release: redis
`

	It("parses a manifest and returns a release-job mapping", func() {
		releaseMapping := NewReleaseMapping(exampleManifest, []string{"red1"})

		Expect(len(releaseMapping["redis"])).To(Equal(1))
		Expect(releaseMapping["redis"][0]).To(Equal("redis-server"))
	})

	It("parses a manifest with two jobs from the same release correctly", func() {
		releaseMapping := NewReleaseMapping(exampleManifest2jobs, []string{"red1", "red2"})

		Expect(len(releaseMapping["redis"])).To(Equal(2))
		Expect(releaseMapping["redis"]).To(ConsistOf("redis-server", "redis-client"))
	})

	It("parses a manifest with two jobs from the same instance group", func() {
		manifest := `---
instance_groups:
- name: red1
  jobs:
  - name: redis-server
    release: redis
- name: red2
  jobs:
  - name: redis-server
    release: redis
  - name: redis-client
    release: redis
`
		releaseMapping := NewReleaseMapping(manifest, []string{"red1", "red2"})

		Expect(len(releaseMapping["redis"])).To(Equal(2))
	})

	It("parses a manifest with two identically-named jobs from different releases", func() {
		manifest := `---
instance_groups:
- name: red1
  jobs:
  - name: redis-server
    release: redis-2.0
- name: red2
  jobs:
  - name: redis-server
    release: redis-2.5
`

		releaseMapping := NewReleaseMapping(manifest, []string{"red1", "red2"})

		Expect(len(releaseMapping["redis-2.5"])).To(Equal(1))
		Expect(len(releaseMapping["redis-2.0"])).To(Equal(1))
	})
})
