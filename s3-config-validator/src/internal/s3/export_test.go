package s3

import (
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
)

type CredIAMProvider = func(c client.ConfigProvider, options ...func(*ec2rolecreds.EC2RoleProvider)) *credentials.Credentials

func SetCredIAMProvider(provider CredIAMProvider) {
	injectableCredIAMProvider = provider
}
