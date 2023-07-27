// Code generated by smithy-go-codegen DO NOT EDIT.

package s3

import (
	"context"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	s3cust "github.com/aws/aws-sdk-go-v2/service/s3/internal/customizations"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Sets an analytics configuration for the bucket (specified by the analytics
// configuration ID). You can have up to 1,000 analytics configurations per bucket.
// You can choose to have storage class analysis export analysis reports sent to a
// comma-separated values (CSV) flat file. See the DataExport request element.
// Reports are updated daily and are based on the object filters that you
// configure. When selecting data export, you specify a destination bucket and an
// optional destination prefix where the file is written. You can export the data
// to a destination bucket in a different account. However, the destination bucket
// must be in the same Region as the bucket that you are making the PUT analytics
// configuration to. For more information, see Amazon S3 Analytics – Storage Class
// Analysis (https://docs.aws.amazon.com/AmazonS3/latest/dev/analytics-storage-class.html)
// . You must create a bucket policy on the destination bucket where the exported
// file is written to grant permissions to Amazon S3 to write objects to the
// bucket. For an example policy, see Granting Permissions for Amazon S3 Inventory
// and Storage Class Analysis (https://docs.aws.amazon.com/AmazonS3/latest/dev/example-bucket-policies.html#example-bucket-policies-use-case-9)
// . To use this operation, you must have permissions to perform the
// s3:PutAnalyticsConfiguration action. The bucket owner has this permission by
// default. The bucket owner can grant this permission to others. For more
// information about permissions, see Permissions Related to Bucket Subresource
// Operations (https://docs.aws.amazon.com/AmazonS3/latest/userguide/using-with-s3-actions.html#using-with-s3-actions-related-to-bucket-subresources)
// and Managing Access Permissions to Your Amazon S3 Resources (https://docs.aws.amazon.com/AmazonS3/latest/userguide/s3-access-control.html)
// . PutBucketAnalyticsConfiguration has the following special errors:
//   - HTTP Error: HTTP 400 Bad Request
//   - Code: InvalidArgument
//   - Cause: Invalid argument.
//   - HTTP Error: HTTP 400 Bad Request
//   - Code: TooManyConfigurations
//   - Cause: You are attempting to create a new configuration but have already
//     reached the 1,000-configuration limit.
//   - HTTP Error: HTTP 403 Forbidden
//   - Code: AccessDenied
//   - Cause: You are not the owner of the specified bucket, or you do not have
//     the s3:PutAnalyticsConfiguration bucket permission to set the configuration on
//     the bucket.
//
// The following operations are related to PutBucketAnalyticsConfiguration :
//   - GetBucketAnalyticsConfiguration (https://docs.aws.amazon.com/AmazonS3/latest/API/API_GetBucketAnalyticsConfiguration.html)
//   - DeleteBucketAnalyticsConfiguration (https://docs.aws.amazon.com/AmazonS3/latest/API/API_DeleteBucketAnalyticsConfiguration.html)
//   - ListBucketAnalyticsConfigurations (https://docs.aws.amazon.com/AmazonS3/latest/API/API_ListBucketAnalyticsConfigurations.html)
func (c *Client) PutBucketAnalyticsConfiguration(ctx context.Context, params *PutBucketAnalyticsConfigurationInput, optFns ...func(*Options)) (*PutBucketAnalyticsConfigurationOutput, error) {
	if params == nil {
		params = &PutBucketAnalyticsConfigurationInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "PutBucketAnalyticsConfiguration", params, optFns, c.addOperationPutBucketAnalyticsConfigurationMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*PutBucketAnalyticsConfigurationOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type PutBucketAnalyticsConfigurationInput struct {

	// The configuration and any analyses for the analytics filter.
	//
	// This member is required.
	AnalyticsConfiguration *types.AnalyticsConfiguration

	// The name of the bucket to which an analytics configuration is stored.
	//
	// This member is required.
	Bucket *string

	// The ID that identifies the analytics configuration.
	//
	// This member is required.
	Id *string

	// The account ID of the expected bucket owner. If the bucket is owned by a
	// different account, the request fails with the HTTP status code 403 Forbidden
	// (access denied).
	ExpectedBucketOwner *string

	noSmithyDocumentSerde
}

type PutBucketAnalyticsConfigurationOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationPutBucketAnalyticsConfigurationMiddlewares(stack *middleware.Stack, options Options) (err error) {
	err = stack.Serialize.Add(&awsRestxml_serializeOpPutBucketAnalyticsConfiguration{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsRestxml_deserializeOpPutBucketAnalyticsConfiguration{}, middleware.After)
	if err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = addHTTPSignerV4Middleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = swapWithCustomHTTPSignerMiddleware(stack, options); err != nil {
		return err
	}
	if err = addOpPutBucketAnalyticsConfigurationValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opPutBucketAnalyticsConfiguration(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = addMetadataRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addPutBucketAnalyticsConfigurationUpdateEndpoint(stack, options); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = v4.AddContentSHA256HeaderMiddleware(stack); err != nil {
		return err
	}
	if err = disableAcceptEncodingGzip(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opPutBucketAnalyticsConfiguration(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		SigningName:   "s3",
		OperationName: "PutBucketAnalyticsConfiguration",
	}
}

// getPutBucketAnalyticsConfigurationBucketMember returns a pointer to string
// denoting a provided bucket member valueand a boolean indicating if the input has
// a modeled bucket name,
func getPutBucketAnalyticsConfigurationBucketMember(input interface{}) (*string, bool) {
	in := input.(*PutBucketAnalyticsConfigurationInput)
	if in.Bucket == nil {
		return nil, false
	}
	return in.Bucket, true
}
func addPutBucketAnalyticsConfigurationUpdateEndpoint(stack *middleware.Stack, options Options) error {
	return s3cust.UpdateEndpoint(stack, s3cust.UpdateEndpointOptions{
		Accessor: s3cust.UpdateEndpointParameterAccessor{
			GetBucketFromInput: getPutBucketAnalyticsConfigurationBucketMember,
		},
		UsePathStyle:                   options.UsePathStyle,
		UseAccelerate:                  options.UseAccelerate,
		SupportsAccelerate:             true,
		TargetS3ObjectLambda:           false,
		EndpointResolver:               options.EndpointResolver,
		EndpointResolverOptions:        options.EndpointOptions,
		UseARNRegion:                   options.UseARNRegion,
		DisableMultiRegionAccessPoints: options.DisableMultiRegionAccessPoints,
	})
}
