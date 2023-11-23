package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

type S3Client struct {
	S3Client *s3.Client
}

func NewS3Client(region, endpoint, id, secret string, useIAMProfile bool, clientOptFns ...func(*s3.Options)) (*S3Client, error) {
	s3client, err := newS3Client(region, endpoint, id, secret, useIAMProfile, clientOptFns...)
	if err != nil {
		return &S3Client{}, err
	}

	return &S3Client{
		S3Client: s3client,
	}, nil
}

// NewS3ClientWithRoleARN
//
// # Warning!
//
// Utilising the assumed role is a highly experimental functionality and is provided as is. Use at your own risk.
func NewS3ClientWithRoleARN(region, endpoint, id, secret, role string, useIAMProfile bool, clientOptFns ...func(*s3.Options)) *S3Client {
	return &S3Client{
		S3Client: newS3ClientWithAssumedRole(region, endpoint, id, secret, role, useIAMProfile, clientOptFns...),
	}
}

func newS3Client(region, endpoint, id, secret string, useIAMProfile bool, fns ...func(*s3.Options)) (client *s3.Client, err error) {
	return newS3ClientWithAssumedRole(region, endpoint, id, secret, "", useIAMProfile, fns...), nil
}

func newS3ClientWithAssumedRole(region, endpoint, id, secret, role string, useIAMProfile bool, fns ...func(*s3.Options)) (client *s3.Client) {
	staticCredentialsProvider := aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(id, secret, ""))

	var creds aws.CredentialsProvider

	if role != "" {
		stsOptions := sts.Options{
			Credentials: staticCredentialsProvider,
			Region:      region,
		}
		//	if endpoint != "" {
		//		stsOptions.EndpointResolver = sts.EndpointResolverFromURL(endpoint)
		//	}
		stsClient := sts.New(stsOptions)
		creds = stscreds.NewAssumeRoleProvider(stsClient, role)
	} else if useIAMProfile {
		creds = aws.NewCredentialsCache(ec2rolecreds.New())
	} else {
		creds = staticCredentialsProvider
	}

	options := s3.Options{
		Credentials: creds,
		Region:      region,
	}

	if endpoint != "" {
		options.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
	}

	return s3.New(options, fns...)
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
	output, err := p.S3Client.GetBucketVersioning(context.TODO(), &s3.GetBucketVersioningInput{
		Bucket: &bucket,
	})
	if err != nil {
		return false, fmt.Errorf("could not check if bucket %s is versioned: %s", bucket, err)
	}

	if output == nil || output.Status != types.BucketVersioningStatusEnabled {
		return false, nil
	}

	return true, nil
}

func (p *S3Client) CanListObjects(bucket string) (err error) {
	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	paginator := s3.NewListObjectsV2Paginator(p.S3Client, params)

	for paginator.HasMorePages() {
		_, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("could not list objects in bucket %s: %s", bucket, err)
		}
	}

	return
}

func (p *S3Client) CanGetObjects(bucket string) (errListObjects error) {
	var cantGetObjects = fmt.Errorf("could not get all objects from bucket %s", bucket)

	params := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	paginator := s3.NewListObjectsV2Paginator(p.S3Client, params)

	for paginator.HasMorePages() {
		listOutput, err := paginator.NextPage(context.TODO())
		if err != nil {
			return cantGetObjects
		}
		for _, content := range listOutput.Contents {
			_, err = p.S3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
				Bucket: aws.String(bucket),
				Key:    content.Key,
			})
			if err != nil {
				return cantGetObjects
			}
		}
	}

	return
}

func (p *S3Client) CanGetObjectVersions(bucket string) (errListObjects error) {
	var cantGetObjectVersions = fmt.Errorf("could not get all object versions from bucket %s", bucket)

	paginator := s3.NewListObjectVersionsPaginator(p.S3Client, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})

	for paginator.HasMorePages() {
		output, err := paginator.NextPage(context.TODO())
		if err != nil {
			return cantGetObjectVersions
		}

		for _, content := range output.Versions {
			_, err = p.S3Client.HeadObject(context.TODO(), &s3.HeadObjectInput{
				Bucket:    aws.String(bucket),
				Key:       content.Key,
				VersionId: content.VersionId,
			})
			if err != nil {
				return cantGetObjectVersions
			}
		}
	}

	return
}

func (p *S3Client) CanPutObjects(bucket string) (err error) {
	fileContent := []byte("Test File, Please delete me if you are reading this")
	fileContentLength := int64(len(fileContent))
	_, err = p.S3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:        aws.String(bucket),
		Key:           aws.String("delete_me"),
		ACL:           types.ObjectCannedACLPrivate,
		Body:          bytes.NewReader(fileContent),
		ContentLength: &fileContentLength,
	})

	if err != nil {
		return fmt.Errorf("could not put object into bucket %s: %s", bucket, err)
	}

	return
}

func (p *S3Client) CanListObjectVersions(bucket string) (err error) {
	paginator := s3.NewListObjectVersionsPaginator(p.S3Client, &s3.ListObjectVersionsInput{
		Bucket: aws.String(bucket),
	})

	for paginator.HasMorePages() {
		_, err := paginator.NextPage(context.TODO())
		if err != nil {
			return fmt.Errorf("could not list object versions in bucket %s: %s", bucket, err)
		}
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
