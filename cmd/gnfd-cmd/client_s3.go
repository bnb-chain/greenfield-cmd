package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/urfave/cli/v2"
)

// NewClient returns a new greenfield client

func NewS3Client(ctx *cli.Context, region string) (*s3.S3, error) {
	configFile := ctx.String("config")
	var config *cmdConfig
	var err error
	if configFile != "" {
		config, err = parseConfigFile(configFile)
		if err != nil {
			return nil, err
		}
	}

	// parsing arguments and setting for connectiong aws service
	awsAccessKeyId := ctx.String("awsAccessKeyId")
	if awsAccessKeyId == "" {
		if config.AwsAccessKeyId == "" {
			return nil, fmt.Errorf("failed to parse aws access key id, please set it in the config file")
		} else {
			awsAccessKeyId = config.AwsAccessKeyId
		}
	}

	awsSecretAccessKey := ctx.String("awsSecretAccessKey")
	if awsSecretAccessKey == "" {
		if config.AwsSecretAccessKey == "" {
			return nil, fmt.Errorf("failed to parse aws secret access key, please set it in the config file")
		} else {
			awsSecretAccessKey = config.AwsSecretAccessKey
		}
	}

	s3Config := &aws.Config{
		Region: aws.String(region),
		Credentials: credentials.NewStaticCredentials(
			awsAccessKeyId,
			awsSecretAccessKey,
			""),
	}

	sess, err := session.NewSession(s3Config)
	if err != nil {
		return nil, fmt.Errorf("Failed to create AWS session: %s", err)
	}

	// create S3 service client
	s3Client := s3.New(sess)
	return s3Client, nil
}
