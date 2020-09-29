package s3

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 . Client

type Client interface {
	IsUnversioned(bucket string) error
	IsVersioned(bucket string) error
	CanListObjects(bucket string) error
	CanListObjectVersions(bucket string) error
	CanGetObjects(bucket string) error
	CanGetObjectVersions(bucket string) error
	CanPutObjects(bucket string) error
}
