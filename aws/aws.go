package aws

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"

	log "github.com/sirupsen/logrus"
)

type AWSDockerCredentialsProvider struct {
	// Region             string
	// AwsAccessKeyID     string
	// AwsSecretAccessKey string
	// AwsSessionToken    string
	session *session.Session
}

func New(region, id, secret, token string) (*AWSDockerCredentialsProvider, error) {
	log.WithFields(log.Fields{
		"provider": "aws",
	}).Debug("Initializing Docker credentials provider")

	if region == "" {
		return nil, errors.New("Must specify an AWS Region")
	}
	config := aws.NewConfig().WithRegion(region)
	if id != "" {
		config = config.WithCredentials(
			credentials.NewStaticCredentials(id, secret, token),
		)
	}

	s, err := session.NewSession(config)
	if err != nil {
		return nil, err
	}

	return &AWSDockerCredentialsProvider{s}, nil
}

func (a *AWSDockerCredentialsProvider) GetCredentials() (string, error) {
	svc := ecr.New(a.session)
	out, err := svc.GetAuthorizationToken(&ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", err
	}

	creds, err := base64.StdEncoding.DecodeString(*out.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", err
	}

	splitCreds := strings.Split(string(creds), ":")
	dockerCreds := struct {
		Username string
		Password string
		Email    string
	}{
		Username: splitCreds[0],
		Password: splitCreds[1],
		Email:    "none",
	}

	bytes, err := json.Marshal(dockerCreds)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}
