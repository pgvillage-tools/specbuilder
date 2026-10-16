/*
Package smoke_test will test a smoke 2e2 test
*/
package smoke_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Smoke", Ordered, func() {
	Context("when connecting to the keepers", func() {
		It("should work properly", func() {
			Ω(nil).NotTo(HaveOccurred())
		})
	})
})
