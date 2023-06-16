package s3_test

import (
	"context"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"

	"github.com/cloudfoundry-incubator/bosh-backup-and-restore/s3-config-validator/src/internal/s3"
)

const (
	VersioningDisabledResponse = `<?xml version="1.0" encoding="UTF-8"?>
<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/"/>`
	VersioningSuspendedResponse = `<?xml version="1.0" encoding="UTF-8"?>
<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <Status>Suspended</Status>
</VersioningConfiguration>`
	VersioningEnabledResponse = `<?xml version="1.0" encoding="UTF-8"?>
<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <Status>Enabled</Status>
</VersioningConfiguration>`
	ListObjectsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ListBucketResult>
    <Name>bucket-name</Name>
    <Prefix></Prefix>
    <Marker></Marker>
    <MaxKeys>1000</MaxKeys>
    <EncodingType>url</EncodingType>
    <IsTruncated>false</IsTruncated>
    <Contents>
        <Key>1.mp4</Key>
        <LastModified>2020-01-01T00:00:00.000Z</LastModified>
        <ETag>etag</ETag>
        <Size>3</Size>
        <Owner>
            <ID>owner-id</ID>
            <DisplayName>Owner</DisplayName>
        </Owner>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
    <Contents>
        <Key>2.mp4</Key>
        <LastModified>2020-01-01T00:00:00.000Z</LastModified>
        <ETag>etag</ETag>
        <Size>6</Size>
        <Owner>
            <ID>owner-id</ID>
            <DisplayName>Owner</DisplayName>
        </Owner>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
    <Contents>
        <Key>3.jpg</Key>
        <LastModified>2020-01-01T00:00:00.000Z</LastModified>
        <ETag>etag</ETag>
        <Size>12</Size>
        <Owner>
            <ID>owner-id</ID>
            <DisplayName>Owner</DisplayName>
        </Owner>
        <StorageClass>STANDARD</StorageClass>
    </Contents>
</ListBucketResult>`
	ListObjectVersionsResponse = `<?xml version="1.0" encoding="UTF-8"?>
<ListVersionsResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/">
    <Name>bucket-name</Name>
    <Prefix></Prefix>
    <KeyMarker></KeyMarker>
    <VersionIdMarker></VersionIdMarker>
    <MaxKeys>1000</MaxKeys>
    <EncodingType>url</EncodingType>
    <IsTruncated>false</IsTruncated>
    <Version>
        <Key>1.mp4</Key>
        <VersionId>version-id</VersionId>
        <IsLatest>true</IsLatest>
        <LastModified>2020-01-01T00:00:00.000Z</LastModified>
        <ETag>etag</ETag>
        <Size>550969170</Size>
        <Owner>
            <ID>owner-id</ID>
            <DisplayName>Owner</DisplayName>
        </Owner>
        <StorageClass>STANDARD</StorageClass>
    </Version>
	<Version>
        <Key>2.mp4</Key>
        <VersionId>version-id</VersionId>
        <IsLatest>true</IsLatest>
        <LastModified>2020-01-01T00:00:00.000Z</LastModified>
        <ETag>etag</ETag>
        <Size>550969170</Size>
        <Owner>
            <ID>owner-id</ID>
            <DisplayName>Owner</DisplayName>
        </Owner>
        <StorageClass>STANDARD</StorageClass>
    </Version>
	<Version>
        <Key>3.jpg</Key>
        <VersionId>version-id</VersionId>
        <IsLatest>true</IsLatest>
        <LastModified>2020-01-01T00:00:00.000Z</LastModified>
        <ETag>etag</ETag>
        <Size>550969170</Size>
        <Owner>
            <ID>owner-id</ID>
            <DisplayName>Owner</DisplayName>
        </Owner>
        <StorageClass>STANDARD</StorageClass>
    </Version>
</ListVersionsResult>`
	AccessDeniedResponse = `<?xml version="1.0" encoding="UTF-8"?>
<Error>
    <Code>AccessDenied</Code>
    <Message>Access Denied</Message>
    <RequestId>request-id</RequestId>
    <HostId>host-id</HostId>
</Error>
`
)

var _ = Describe("S3Client", func() {
	var creds aws.CredentialsProvider

	Describe("given an S3 server", func() {
		var (
			fakeS3Server       *ghttp.Server
			credSpy            func(options *awss3.Options)
			fakeS3ServerConfig func(options *awss3.Options)
		)

		BeforeEach(func() {
			fakeS3Server = ghttp.NewServer()
			fakeS3ServerConfig = func(options *awss3.Options) {
				options.EndpointResolver = awss3.EndpointResolverFromURL(fakeS3Server.URL(), func(endpoint *aws.Endpoint) {
					endpoint.HostnameImmutable = true
				})
			}
			credSpy = func(o *awss3.Options) {
				creds = o.Credentials
			}
		})

		AfterEach(func() {
			fakeS3Server.Close()
		})

		When("creating an s3client", func() {
			When("we are not using an IAMProfile", func() {
				It("uses static credentials", func() {
					_, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, credSpy, fakeS3ServerConfig)
					Expect(err).NotTo(HaveOccurred())

					actualCreds, err := creds.Retrieve(context.TODO())
					Expect(err).ToNot(HaveOccurred())
					Expect(actualCreds.SecretAccessKey).To(Equal("test-secret"))
					Expect(actualCreds.AccessKeyID).To(Equal("test-id"))
					Expect(actualCreds.Source).To(Equal("StaticCredentials"))
				})
			})

			When("we are using an IAMProfile", func() {
				It("uses EC2 Role credentials", func() {
					_, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", true, credSpy, fakeS3ServerConfig)
					Expect(err).NotTo(HaveOccurred())

					_, err = creds.Retrieve(context.TODO())
					Expect(err).To(MatchError(ContainSubstring("operation error ec2imds: GetMetadata")))
				})
			})
		})

		Context("Bucket Versioning", func() {
			When("I can get a bucket's versioning", func() {
				Context("bucket has never been versioned", func() {
					BeforeEach(func() {
						fakeS3Server.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/test-bucket", "versioning"),
								ghttp.RespondWith(http.StatusOK, VersioningDisabledResponse),
							),
						)
					})

					It("IsUnversioned succeeds", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsUnversioned("test-bucket")).To(Succeed())
					})

					It("IsVersioned returns an error", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsVersioned("test-bucket")).To(Not(Succeed()))
					})
				})

				Context("bucket is not versioned", func() {
					BeforeEach(func() {
						fakeS3Server.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/test-bucket", "versioning"),
								ghttp.RespondWith(http.StatusOK, VersioningSuspendedResponse),
							),
						)
					})

					It("IsUnversioned succeeds", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsUnversioned("test-bucket")).To(Succeed())
					})

					It("IsVersioned returns an error", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsVersioned("test-bucket")).To(Not(Succeed()))
					})
				})

				Context("bucket is versioned", func() {
					BeforeEach(func() {
						fakeS3Server.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/test-bucket", "versioning"),
								ghttp.RespondWith(http.StatusOK, VersioningEnabledResponse),
							),
						)
					})

					It("IsUnversioned returns an error", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsUnversioned("test-bucket")).To(Not(Succeed()))
					})

					It("IsVersioned succeeds", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsVersioned("test-bucket")).To(Succeed())
					})

				})
			})

			When("I can not get a bucket's versioning", func() {
				Context("call fails", func() {
					BeforeEach(func() {
						fakeS3Server.AppendHandlers(
							ghttp.CombineHandlers(
								ghttp.VerifyRequest("GET", "/test-bucket", "versioning"),
								ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
							),
						)
					})

					It("IsUnversioned returns an error", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsUnversioned("test-bucket")).To(MatchError("could not check if bucket test-bucket is versioned: operation error S3: GetBucketVersioning, https response error StatusCode: 403, RequestID: request-id, HostID: host-id, api error AccessDenied: Access Denied"))
					})

					It("IsVersioned returns an error", func() {
						probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
						Expect(err).ToNot(HaveOccurred())

						Expect(probe.IsVersioned("test-bucket")).To(MatchError("could not check if bucket test-bucket is versioned: operation error S3: GetBucketVersioning, https response error StatusCode: 403, RequestID: request-id, HostID: host-id, api error AccessDenied: Access Denied"))
					})
				})
			})
		})

		Context("List Objects", func() {
			When("I can list objects in a bucket", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket"),
							ghttp.RespondWith(http.StatusOK, ListObjectsResponse),
						),
					)
				})

				It("returns no error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanListObjects("test-bucket")).To(Succeed())
				})
			})

			When("I can not list objects in a bucket", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanListObjects("test-bucket")).To(MatchError("could not list objects in bucket test-bucket: operation error S3: ListObjectsV2, https response error StatusCode: 403, RequestID: request-id, HostID: host-id, api error AccessDenied: Access Denied"))
				})
			})
		})

		Context("List Object Versions", func() {
			When("I can list object versions in a bucket", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket", "versions"),
							ghttp.RespondWith(http.StatusOK, ListObjectVersionsResponse),
						),
					)
				})

				It("returns no error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanListObjectVersions("test-bucket")).To(Succeed())
				})
			})

			When("I can not list object versions in a bucket", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket", "versions"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanListObjectVersions("test-bucket")).To(MatchError("could not list object versions in bucket test-bucket: operation error S3: ListObjectVersions, https response error StatusCode: 403, RequestID: request-id, HostID: host-id, api error AccessDenied: Access Denied"))
				})
			})
		})

		Context("Get Object", func() {
			When("I can get all objects' meta data", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket"),
							ghttp.RespondWith(http.StatusOK, ListObjectsResponse),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/1.mp4"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/2.mp4"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/3.jpg"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
					)
				})

				It("returns no error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanGetObjects("test-bucket")).To(Succeed())
				})
			})

			When("I can not get all objects' meta data", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket"),
							ghttp.RespondWith(http.StatusOK, ListObjectsResponse),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/1.mp4"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/2.mp4"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/3.jpg"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanGetObjects("test-bucket")).To(MatchError("could not get all objects from bucket test-bucket"))
				})
			})

			When("I can not get all objects' meta data when list object fails", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanGetObjects("test-bucket")).To(MatchError("could not get all objects from bucket test-bucket"))
				})
			})
		})

		Context("Get Object Versions", func() {
			When("I can get all object versions", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket", "versions"),
							ghttp.RespondWith(http.StatusOK, ListObjectVersionsResponse),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/1.mp4", "versionId=version-id"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/2.mp4", "versionId=version-id"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/3.jpg", "versionId=version-id"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
					)
				})

				It("returns no error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanGetObjectVersions("test-bucket")).To(Succeed())
				})
			})

			When("I can not get all object versions", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket", "versions"),
							ghttp.RespondWith(http.StatusOK, ListObjectVersionsResponse),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/1.mp4"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/2.mp4"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("HEAD", "/test-bucket/3.jpg"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanGetObjectVersions("test-bucket")).To(MatchError("could not get all object versions from bucket test-bucket"))
				})
			})

			When("I can not get all object versions when list object versions fails", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("GET", "/test-bucket", "versions"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanGetObjectVersions("test-bucket")).To(MatchError("could not get all object versions from bucket test-bucket"))
				})
			})
		})

		Context("Put Object", func() {
			When("I can put an object", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/test-bucket/delete_me"),
							ghttp.RespondWith(http.StatusOK, ""),
						),
					)
				})

				It("returns no error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanPutObjects("test-bucket")).To(Succeed())
				})
			})

			When("I can not put an object", func() {
				BeforeEach(func() {
					fakeS3Server.AppendHandlers(
						ghttp.CombineHandlers(
							ghttp.VerifyRequest("PUT", "/test-bucket/delete_me"),
							ghttp.RespondWith(http.StatusForbidden, AccessDeniedResponse),
						),
					)
				})

				It("returns an error", func() {
					probe, err := s3.NewS3Client("test-region", fakeS3Server.URL(), "test-id", "test-secret", false, fakeS3ServerConfig)
					Expect(err).ToNot(HaveOccurred())

					Expect(probe.CanPutObjects("test-bucket")).To(MatchError("could not put object into bucket test-bucket: operation error S3: PutObject, https response error StatusCode: 403, RequestID: request-id, HostID: host-id, api error AccessDenied: Access Denied"))
				})
			})
		})
	})
})
