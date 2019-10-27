package cheops

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
)

type AWSDockerCredentialsProvider struct {
	Region string
}

func (a AWSDockerCredentialsProvider) GetCredentials() (string, error) {
	session, err := session.NewSession(aws.NewConfig().WithRegion(a.Region))

	if err != nil {
		return "", err
	}

	svc := ecr.New(session)
	out, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}

	return *out.AuthorizationData[0].AuthorizationToken, nil
}
