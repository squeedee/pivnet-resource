package check_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/pivotal-cf-experimental/go-pivnet"
	"github.com/pivotal-cf-experimental/pivnet-resource/check"
	"github.com/pivotal-cf-experimental/pivnet-resource/check/checkfakes"
	"github.com/pivotal-cf-experimental/pivnet-resource/concourse"
	"github.com/pivotal-cf-experimental/pivnet-resource/versions"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Check", func() {
	var (
		fakeFilter       *checkfakes.FakeFilter
		fakePivnetClient *checkfakes.FakePivnetClient
		fakeSorter       *checkfakes.FakeSorter

		checkRequest concourse.CheckRequest
		checkCommand *check.CheckCommand

		releaseTypes    []string
		releaseTypesErr error

		allReleases      []pivnet.Release
		releasesErr      error
		filteredReleases []pivnet.Release

		releasesByReleaseTypeErr error
		releasesByVersionErr     error
		etagErr                  error
		logging                  *log.Logger

		tempDir     string
		logFilePath string
	)

	BeforeEach(func() {
		fakeFilter = &checkfakes.FakeFilter{}
		fakePivnetClient = &checkfakes.FakePivnetClient{}
		fakeSorter = &checkfakes.FakeSorter{}
		logging = log.New(ioutil.Discard, "doesn't matter", 0)

		releasesByReleaseTypeErr = nil
		releasesByVersionErr = nil
		releaseTypesErr = nil
		releasesErr = nil
		etagErr = nil

		releaseTypes = []string{
			"foo release",
			"bar",
			"third release type",
		}

		allReleases = []pivnet.Release{
			{
				ID:          1,
				Version:     "A",
				ReleaseType: releaseTypes[0],
			},
			{
				ID:          2,
				Version:     "C",
				ReleaseType: releaseTypes[1],
			},
			{
				ID:          3,
				Version:     "B",
				ReleaseType: releaseTypes[2],
			},
		}

		filteredReleases = allReleases

		checkRequest = concourse.CheckRequest{
			Source: concourse.Source{
				APIToken:    "some-api-token",
				ProductSlug: productSlug,
			},
		}

		var err error
		tempDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		logFilePath = filepath.Join(tempDir, "pivnet-resource-check.log1234")
		err = ioutil.WriteFile(logFilePath, []byte("initial log content"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())
	})

	JustBeforeEach(func() {
		fakePivnetClient.ReleaseTypesReturns(releaseTypes, releaseTypesErr)
		fakePivnetClient.ReleasesForProductSlugReturns(allReleases, releasesErr)

		fakePivnetClient.ReleaseETagStub = func(productSlug string, releaseID int) (string, error) {
			etag := fmt.Sprintf("etag-%d", releaseID)
			return etag, etagErr
		}

		fakeFilter.ReleasesByReleaseTypeReturns(filteredReleases, releasesByReleaseTypeErr)
		fakeFilter.ReleasesByVersionReturns(filteredReleases, releasesByVersionErr)

		binaryVersion := "v0.1.2-unit-tests"

		checkCommand = check.NewCheckCommand(
			logging,
			binaryVersion,
			fakeFilter,
			fakePivnetClient,
			fakeSorter,
			logFilePath,
		)
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("returns the most recent version without error", func() {
		response, err := checkCommand.Run(checkRequest)
		Expect(err).NotTo(HaveOccurred())

		expectedVersionWithEtag, err := versions.CombineVersionAndETag(
			allReleases[0].Version, fmt.Sprintf("etag-%d", allReleases[0].ID),
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(response).To(HaveLen(1))
		Expect(response[0].ProductVersion).To(Equal(expectedVersionWithEtag))
	})

	Context("when no releases are returned", func() {
		BeforeEach(func() {
			allReleases = []pivnet.Release{}
		})

		It("returns empty response without error", func() {
			response, err := checkCommand.Run(checkRequest)
			Expect(err).NotTo(HaveOccurred())

			Expect(response).To(BeEmpty())
		})
	})

	Context("when log files already exist", func() {
		var (
			otherFilePath1 string
			otherFilePath2 string
		)

		BeforeEach(func() {
			otherFilePath1 = filepath.Join(tempDir, "pivnet-resource-check.log1")
			err := ioutil.WriteFile(otherFilePath1, []byte("initial log content"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			otherFilePath2 = filepath.Join(tempDir, "pivnet-resource-check.log2")
			err = ioutil.WriteFile(otherFilePath2, []byte("initial log content"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})

		It("removes the other log files", func() {
			_, err := checkCommand.Run(checkRequest)
			Expect(err).NotTo(HaveOccurred())

			_, err = os.Stat(otherFilePath1)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(otherFilePath2)
			Expect(err).To(HaveOccurred())
			Expect(os.IsNotExist(err)).To(BeTrue())

			_, err = os.Stat(logFilePath)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Context("when there is an error getting release types", func() {
		BeforeEach(func() {
			releaseTypesErr = fmt.Errorf("some error")
		})

		It("returns an error", func() {
			_, err := checkCommand.Run(checkRequest)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("some error"))
		})
	})

	Context("when there is an error getting releases", func() {
		BeforeEach(func() {
			releasesErr = fmt.Errorf("some error")
		})

		It("returns an error", func() {
			_, err := checkCommand.Run(checkRequest)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("some error"))
		})
	})

	Context("when there is an error getting etag", func() {
		BeforeEach(func() {
			etagErr = fmt.Errorf("some error")
		})

		It("returns an error", func() {
			_, err := checkCommand.Run(checkRequest)
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("some error"))
		})
	})

	Describe("when a version is provided", func() {
		Context("when the version is the latest", func() {
			BeforeEach(func() {
				versionWithETag, err := versions.CombineVersionAndETag(
					allReleases[0].Version, fmt.Sprintf("etag-%d", allReleases[0].ID),
				)
				Expect(err).NotTo(HaveOccurred())

				checkRequest.Version = concourse.Version{
					ProductVersion: versionWithETag,
				}
			})

			It("returns the most recent version", func() {
				response, err := checkCommand.Run(checkRequest)
				Expect(err).NotTo(HaveOccurred())

				versionWithETagA, err := versions.CombineVersionAndETag(
					allReleases[0].Version, fmt.Sprintf("etag-%d", allReleases[0].ID),
				)
				Expect(err).NotTo(HaveOccurred())

				Expect(response).To(HaveLen(1))
				Expect(response[0].ProductVersion).To(Equal(versionWithETagA))
			})
		})

		Context("when the version is not the latest", func() {
			BeforeEach(func() {
				versionWithETag, err := versions.CombineVersionAndETag(
					allReleases[2].Version, fmt.Sprintf("etag-%d", allReleases[2].ID),
				)
				Expect(err).NotTo(HaveOccurred())

				checkRequest.Version = concourse.Version{
					ProductVersion: versionWithETag,
				}
			})

			It("returns the most recent version", func() {
				response, err := checkCommand.Run(checkRequest)
				Expect(err).NotTo(HaveOccurred())

				versionWithETagC, err := versions.CombineVersionAndETag(
					allReleases[1].Version, fmt.Sprintf("etag-%d", allReleases[1].ID),
				)
				Expect(err).NotTo(HaveOccurred())

				versionWithETagA, err := versions.CombineVersionAndETag(
					allReleases[0].Version, fmt.Sprintf("etag-%d", allReleases[0].ID),
				)
				Expect(err).NotTo(HaveOccurred())

				Expect(response).To(HaveLen(2))
				Expect(response[0].ProductVersion).To(Equal(versionWithETagC))
				Expect(response[1].ProductVersion).To(Equal(versionWithETagA))
			})
		})
	})

	Context("when the release type is specified", func() {
		BeforeEach(func() {
			checkRequest.Source.ReleaseType = releaseTypes[1]

			filteredReleases = []pivnet.Release{allReleases[1]}
		})

		It("returns the most recent version with that release type", func() {
			response, err := checkCommand.Run(checkRequest)
			Expect(err).NotTo(HaveOccurred())

			versionWithETagC, err := versions.CombineVersionAndETag(
				allReleases[1].Version, fmt.Sprintf("etag-%d", allReleases[1].ID),
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(response).To(HaveLen(1))
			Expect(response[0].ProductVersion).To(Equal(versionWithETagC))
		})

		Context("when the release type is invalid", func() {
			BeforeEach(func() {
				checkRequest.Source.ReleaseType = "not a valid release type"
			})

			It("returns an error", func() {
				_, err := checkCommand.Run(checkRequest)
				Expect(err).To(HaveOccurred())

				Expect(err.Error()).To(MatchRegexp(".*release_type.*one of"))
				Expect(err.Error()).To(ContainSubstring(releaseTypes[0]))
				Expect(err.Error()).To(ContainSubstring(releaseTypes[1]))
				Expect(err.Error()).To(ContainSubstring(releaseTypes[2]))
			})
		})

		Context("when filtering returns an error", func() {
			BeforeEach(func() {
				releasesByReleaseTypeErr = fmt.Errorf("some release type error")
			})

			It("returns the error", func() {
				_, err := checkCommand.Run(checkRequest)
				Expect(err).To(HaveOccurred())

				Expect(err).To(Equal(releasesByReleaseTypeErr))
			})
		})
	})

	Context("when the product version is specified", func() {
		BeforeEach(func() {
			checkRequest.Source.ReleaseType = releaseTypes[1]

			filteredReleases = []pivnet.Release{allReleases[1]}
		})

		BeforeEach(func() {
			checkRequest.Source.ProductVersion = "C"
		})

		It("returns the newest release with that version without error", func() {
			response, err := checkCommand.Run(checkRequest)
			Expect(err).NotTo(HaveOccurred())

			versionWithETagC, err := versions.CombineVersionAndETag(
				allReleases[1].Version, fmt.Sprintf("etag-%d", allReleases[1].ID),
			)
			Expect(err).NotTo(HaveOccurred())

			Expect(response).To(HaveLen(1))
			Expect(response[0].ProductVersion).To(Equal(versionWithETagC))
		})

		Context("when filtering returns an error", func() {
			BeforeEach(func() {
				releasesByVersionErr = fmt.Errorf("some version error")
			})

			It("returns the error", func() {
				_, err := checkCommand.Run(checkRequest)
				Expect(err).To(HaveOccurred())

				Expect(err).To(Equal(releasesByVersionErr))
			})
		})
	})

	Context("when sorting by semver", func() {
		var (
			semverOrderedReleases []pivnet.Release
		)

		BeforeEach(func() {
			checkRequest.Source.SortBy = concourse.SortBySemver
			checkRequest.Version = concourse.Version{
				ProductVersion: "1.2.3#etag-5432",
			}

			semverOrderedReleases = []pivnet.Release{
				{
					ID:      7654,
					Version: "2.3.4",
				},
				{
					ID:      6543,
					Version: "1.2.4",
				},
				{
					ID:      5432,
					Version: "1.2.3",
				},
			}

			fakeSorter.SortBySemverReturns(semverOrderedReleases, nil)
		})

		It("returns in ascending semver order", func() {
			response, err := checkCommand.Run(checkRequest)
			Expect(err).NotTo(HaveOccurred())

			versionsWithETags := make([]string, len(semverOrderedReleases))

			versionsWithETags[0], err = versions.CombineVersionAndETag(
				"1.2.4", fmt.Sprintf("etag-%d", 6543),
			)
			versionsWithETags[1], err = versions.CombineVersionAndETag(
				"2.3.4", fmt.Sprintf("etag-%d", 7654),
			)

			Expect(response).To(HaveLen(2))
			Expect(response[0].ProductVersion).To(Equal(versionsWithETags[0]))
			Expect(response[1].ProductVersion).To(Equal(versionsWithETags[1]))
			Expect(fakeSorter.SortBySemverCallCount()).To(Equal(1))
		})

		Context("when sorting by semver returns an error", func() {
			var (
				semverErr error
			)

			BeforeEach(func() {
				semverErr = errors.New("semver error")

				fakeSorter.SortBySemverReturns(nil, semverErr)
			})

			It("returns error", func() {
				_, err := checkCommand.Run(checkRequest)
				Expect(err).To(HaveOccurred())

				Expect(err).To(Equal(semverErr))
			})
		})
	})
})
