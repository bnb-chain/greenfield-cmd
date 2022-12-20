package main

import (
	"context"
	"fmt"
	"log"

	"github.com/urfave/cli/v2"
)

// cmdPutObj return the command to finish second stage of putObject
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
$ bfs  pre-mb s3://bucketname`,
	}
}

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
$ bfs mb s3://bucketname`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "public",
				Value: false,
				Usage: "indicate whether the bucket is public",
			},
		},
	}
}

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

	isPublic := ctx.Bool("public")

	if err = s3Client.CreateBucket(c, bucketName, isPublic); err != nil {
		return err
	}
	fmt.Println("create bucket succ")
	return nil
}

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

	signature, _, err := s3Client.GetApproval(c, bucketName, "")
	if err != nil {
		return err
	}

	fmt.Printf("get signature:", signature)
	return nil
}

func getBucketName(ctx *cli.Context) (string, error) {
	if ctx.NArg() < 1 {
		return "", fmt.Errorf("the args should be more than 1")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName := ParseBucket(urlInfo)

	if bucketName == "" {
		return "", fmt.Errorf("fail to parse bucketname")
	}
	return bucketName, nil
}
