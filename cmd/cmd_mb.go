package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	spClient "github.com/bnb-chain/gnfd-go-sdk/client/sp"
	"github.com/urfave/cli/v2"
)

// cmdPreMakeBucket get approval of creating bucket from the storage provider
func cmdPreMakeBucket() *cli.Command {
	return &cli.Command{
		Name:      "pre-mb",
		Action:    preCreateBucket,
		Usage:     "pre make bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
 preMakeBucket and get approval from storage provider

Examples:
# the first phase of putObject
$ gnfd  pre-mb gnfd://bucketname`,
	}
}

// cmdMakeBucket create a new Bucket
func cmdMakeBucket() *cli.Command {
	return &cli.Command{
		Name:      "mb",
		Action:    createBucket,
		Usage:     "create a new bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Create a new bucket and set a createBucketMsg to storage provider, the bucket name should  unique and the default acl is Public

Examples:
# Create a new bucket
$ gnfd mb gnfd://bucketname`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "public",
				Value: false,
				Usage: "indicate whether the bucket is public",
			},
		},
	}
}

// createBucket send the create bucket api to storage provider
func createBucket(ctx *cli.Context) error {
	bucketName, err := getBucketName(ctx)
	if err != nil {
		return err
	}

	s3Client, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	if err = s3Client.CreateBucket(c, bucketName, spClient.NewAuthInfo(false, "")); err != nil {
		return err
	}
	fmt.Println("create bucket succ")
	return nil
}

// preCreateBucket send the request to sp to get approval of creating bucket
func preCreateBucket(ctx *cli.Context) error {
	bucketName, err := getBucketName(ctx)
	if err != nil {

		return err
	}

	s3Client, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	signature, err := s3Client.GetApproval(c, bucketName, "", spClient.NewAuthInfo(false, ""))
	if err != nil {
		return err
	}

	fmt.Printf("get signature:", signature)
	return nil
}

func getBucketName(ctx *cli.Context) (string, error) {
	if ctx.NArg() < 1 {
		return "", errors.New("the args should be more than 1")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName := ParseBucket(urlInfo)

	if bucketName == "" {
		return "", errors.New("fail to parse bucketname")
	}
	return bucketName, nil
}
