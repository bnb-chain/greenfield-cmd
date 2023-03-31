package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v2"
)

// cmdHeadObj return the command to finish uploading payload of the object
func cmdHeadObj() *cli.Command {
	return &cli.Command{
		Name:      "head-obj",
		Action:    headObject,
		Usage:     "query object info",
		ArgsUsage: "OBJECT-URL",
		Description: `
send headObject txn to chain and fetch objectInfo on greenfield chain
Examples:
$ gnfd-cmd head-bucket gnfd://bucket-name/object-name`,
	}
}

// cmdHeadBucket return the command to finish uploading payload of the object
func cmdHeadBucket() *cli.Command {
	return &cli.Command{
		Name:      "head-bucket",
		Action:    headBucket,
		Usage:     "query bucket info",
		ArgsUsage: "BUCKET-URL",
		Description: `
send headBucket txn to chain and fetch bucketInfo on greenfield chain
Examples:
$ gnfd-cmd head-bucket gnfd://bucket-name`,
	}
}

func headObject(ctx *cli.Context) error {
	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadObject := context.WithCancel(globalContext)
	defer cancelHeadObject()

	objectInfo, err := client.HeadObject(c, bucketName, objectName)
	if err != nil {
		fmt.Println("no such object")
		return nil
	}
	parseChainInfo(objectInfo.String(), false)
	fmt.Println("object status:", objectInfo.ObjectStatus.String())
	return nil
}

// headBucket send the create bucket request to storage provider
func headBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelHeadBucket := context.WithCancel(globalContext)
	defer cancelHeadBucket()

	bucketInfo, err := client.HeadBucket(c, bucketName)
	if err != nil {
		fmt.Println("no such bucket")
		return nil
	}

	parseChainInfo(bucketInfo.String(), true)
	return nil
}
