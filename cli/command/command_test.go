package command

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("bbr", func() {
	Describe("ExtractNameFromAddress", func() {
		It("Returns IP address when that's all it's given", func() {
			Expect(extractNameFromAddress("10.5.26.522")).To(Equal("10.5.26.522"))
		})

		It("Returns IP address when it's given IP and port", func() {
			Expect(extractNameFromAddress("10.5.26.522:53")).To(Equal("10.5.26.522"))
		})

		It("Returns IP address when it's given IP, protocol and port", func() {
			Expect(extractNameFromAddress("https://10.5.26.522:53")).To(Equal("10.5.26.522"))
		})

		It("Returns IP address when it's given IP and protocol", func() {
			Expect(extractNameFromAddress("https://10.5.26.522")).To(Equal("10.5.26.522"))
		})

		It("Returns hostname when that's all it's given", func() {
			Expect(extractNameFromAddress("my.bosh.com")).To(Equal("my.bosh.com"))
		})

		It("Returns hostname when it's given hostname and port", func() {
			Expect(extractNameFromAddress("my.bosh.com:42")).To(Equal("my.bosh.com"))
		})

		It("Returns hostname when it's given hostname, protocol and port", func() {
			Expect(extractNameFromAddress("http://my.bosh.com:42")).To(Equal("my.bosh.com"))
		})

		It("Returns hostname when it's given hostname and protocol", func() {
			Expect(extractNameFromAddress("http://my.bosh.com")).To(Equal("my.bosh.com"))
		})
	})
})
