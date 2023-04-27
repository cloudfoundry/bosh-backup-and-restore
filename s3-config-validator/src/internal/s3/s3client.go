package s3

import (
	"bytes"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type S3Client struct {
	S3Client *s3.S3
}

func NewS3Client(region, endpoint, id, secret string, useIAMProfile bool) (*S3Client, error) {
	s3client, err := newS3Client(region, endpoint, id, secret, useIAMProfile)
	if err != nil {
		return &S3Client{}, err
	}

	return &S3Client{
		S3Client: s3client,
	}, nil
}

var injectableCredIAMProvider = ec2rolecreds.NewCredentials

func newS3Client(region, endpoint, id, secret string, useIAMProfile bool) (client *s3.S3, err error) {
	creds := credentials.NewStaticCredentials(id, secret, "")

	if useIAMProfile {
		intermediateSession, err := session.NewSession(aws.NewConfig().WithRegion(region))
		if err != nil {
			return nil, err
		}

		creds = injectableCredIAMProvider(intermediateSession)
	}

	session, err := session.NewSession(&aws.Config{
		Region:           &region,
		Credentials:      creds,
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(true),
	})

	return s3.New(session), err
}

func (p *S3Client) IsUnversioned(bucket string) error {
	isVersioned, err := p.getBucketVersioning(bucket)
	if err != nil {
		return err
	}

	if isVersioned {
		return fmt.Errorf("bucket %s is versioned", bucket)
	}

	return nil
}

func (p *S3Client) getBucketVersioning(bucket string) (isVersioned bool, err error) {
	output, err := p.S3Client.GetBucketVersioning(&s3.GetBucketVersioningInput{
		Bucket: &bucket,
	})
	if err != nil {
		return false, fmt.Errorf("could not check if bucket %s is versioned: %s", bucket, err)
	}

	if output == nil || output.Status == nil || *output.Status != "Enabled" {
		return false, nil
	}

	return true, nil
}

func (p *S3Client) CanListObjects(bucket string) (err error) {
	err = p.S3Client.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(""),
	}, func(output *s3.ListObjectsOutput, lastPage bool) bool {
		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("could not list objects in bucket %s: %s", bucket, err)
	}

	return
}

func (p *S3Client) CanGetObjects(bucket string) (errListObjects error) {
	canFetchAllFiles := true

	errListObjects = p.S3Client.ListObjectsPages(&s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(""),
	}, func(output *s3.ListObjectsOutput, lastPage bool) bool {
		for _, content := range output.Contents {
			_, errGetObject := p.S3Client.HeadObject(&s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    content.Key,
			})
			if errGetObject != nil {
				canFetchAllFiles = false
			}
		}

		return !lastPage
	})

	if !canFetchAllFiles || errListObjects != nil {
		return fmt.Errorf("could not get all objects from bucket %s", bucket)
	}

	return nil
}

func (p *S3Client) CanGetObjectVersions(bucket string) (errListObjects error) {
	canFetchAllFiles := true

	errListObjects = p.S3Client.ListObjectVersionsPages(&s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}, func(output *s3.ListObjectVersionsOutput, lastPage bool) bool {
		for _, content := range output.Versions {
			_, errGetObject := p.S3Client.HeadObject(&s3.HeadObjectInput{
				Bucket:    aws.String(bucket),
				Key:       content.Key,
				VersionId: content.VersionId,
			})
			if errGetObject != nil {
				canFetchAllFiles = false
			}
		}

		return !lastPage
	})

	if !canFetchAllFiles || errListObjects != nil {
		return fmt.Errorf("could not get all object versions from bucket %s", bucket)
	}

	return nil
}

func (p *S3Client) CanPutObjects(bucket string) (err error) {
	fileContent := []byte("Test File, Please delete me if you are reading this")
	_, err = p.S3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String("delete_me"),
		ACL:           aws.String("private"),
		Body:          bytes.NewReader(fileContent),
		ContentLength: aws.Int64(int64(len(fileContent))),
	})

	if err != nil {
		return fmt.Errorf("could not put object into bucket %s: %s", bucket, err)
	}

	return
}

func (p *S3Client) CanListObjectVersions(bucket string) (err error) {
	err = p.S3Client.ListObjectVersionsPages(&s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	}, func(output *s3.ListObjectVersionsOutput, lastPage bool) bool {
		return !lastPage
	})

	if err != nil {
		return fmt.Errorf("could not list object versions in bucket %s: %s", bucket, err)
	}

	return
}

func (p *S3Client) IsVersioned(bucket string) (err error) {
	isVersioned, err := p.getBucketVersioning(bucket)
	if err != nil {
		return err
	}

	if isVersioned {
		return nil
	}

	return fmt.Errorf("bucket %s is unversioned", bucket)
}
