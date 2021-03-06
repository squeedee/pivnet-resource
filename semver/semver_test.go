package semver_test

import (
	"io/ioutil"
	"log"

	bsemver "github.com/blang/semver"
	"github.com/pivotal-cf-experimental/pivnet-resource/semver"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("SemverConverter", func() {
	var (
		s *semver.SemverConverter
	)

	BeforeEach(func() {
		testLogger := log.New(ioutil.Discard, "it doesn't matter", 0)
		s = semver.NewSemverConverter(testLogger)
	})

	Describe("ToValidSemver", func() {
		var (
			input string
		)

		BeforeEach(func() {
			input = "1.2.3-edge.12"
		})

		It("parses valid semver", func() {
			returned, err := s.ToValidSemver(input)
			Expect(err).NotTo(HaveOccurred())

			expectedReturned := bsemver.Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
				Pre: []bsemver.PRVersion{
					{VersionStr: "edge"},
					{VersionNum: 12, IsNum: true},
				},
			}
			Expect(returned).To(Equal(expectedReturned))
		})

		Context("when parsing a version as semver fails", func() {
			BeforeEach(func() {
				input = "invalid-semver"
			})

			It("returns error", func() {
				_, err := s.ToValidSemver(input)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the input has one component", func() {
			BeforeEach(func() {
				input = "1"
			})

			It("returns with 3 components (adds zeros) without error", func() {
				returned, err := s.ToValidSemver(input)
				Expect(err).NotTo(HaveOccurred())

				expectedReturned := bsemver.Version{
					Major: 1,
					Minor: 0,
					Patch: 0,
				}
				Expect(returned).To(Equal(expectedReturned))
			})
		})

		Context("when the input has two components", func() {
			BeforeEach(func() {
				input = "1.2"
			})

			It("returns with 3 components (adds zeros) without error", func() {
				returned, err := s.ToValidSemver(input)
				Expect(err).NotTo(HaveOccurred())

				expectedReturned := bsemver.Version{
					Major: 1,
					Minor: 2,
					Patch: 0,
				}
				Expect(returned).To(Equal(expectedReturned))
			})
		})

		Context("when a version has more than 3 components", func() {
			BeforeEach(func() {
				input = "1.2.3.4"
			})

			It("returns error", func() {
				_, err := s.ToValidSemver(input)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
