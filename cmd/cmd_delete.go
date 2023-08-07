package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bnb-chain/greenfield-go-sdk/client"
	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdDelBucket delete an existed Bucket, the bucket must be empty
func cmdDelBucket() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Action:    deleteBucket,
		Usage:     "delete an existed bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Send a deleteBucket txn to greenfield chain, the bucket must be empty before deleting

Examples:
# Delete an existed bucket called gnfd-bucket
$ gnfd-cmd bucket delete gnfd://gnfd-bucket/gnfd-object`,
	}
}

// cmdDelObject delete an existed object in bucket
func cmdDelObject() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Action:    deleteObject,
		Usage:     "delete an existed object",
		ArgsUsage: "BUCKET-URL",
		Description: `
Send a deleteObject txn to greenfield chain

Examples:
# Delete an existed object called gnfd-object
$ gnfd-cmd object delete gnfd://gnfd-bucket/gnfd-object`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  recursiveFlag,
				Value: false,
				Usage: "performed on all files or objects under the specified directory or prefix in a recursive way",
			},
		},
	}
}

// cmdDelGroup delete an existed group
func cmdDelGroup() *cli.Command {
	return &cli.Command{
		Name:      "delete",
		Action:    deleteGroup,
		Usage:     "delete an existed group",
		ArgsUsage: "GROUP-NAME",
		Description: `
Send a deleteGroup txn to greenfield chain

Examples:
# Delete an existed group
$ gnfd-cmd group delete group-name`,
	}
}

// deleteBucket send the deleteBucket msg to greenfield
func deleteBucket(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number more than one"))
	}
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelDelBucket := context.WithCancel(globalContext)
	defer cancelDelBucket()

	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		fmt.Printf("bucket %s not exist or already deleted\n", bucketName)
	}

	txnHash, err := client.DeleteBucket(c, bucketName, sdktypes.DeleteBucketOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		fmt.Println("delete bucket error:", err.Error())
		return nil
	}

	err = waitTxnStatus(client, c, txnHash, "DeleteBucket")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("delete bucket: %s successfully, txn hash: %s\n", bucketName, txnHash)
	return nil
}

// deleteObject send the deleteBucket msg to greenfield
func deleteObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number more than one"))
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelDelObject := context.WithCancel(globalContext)
	defer cancelDelObject()
	supportRecursive := ctx.Bool(recursiveFlag)

	// if it is a folder and set the --recursive flag , list all the objects and delete them one by one
	if supportRecursive {
		foldName := objectName
		if !strings.HasSuffix(foldName, "/") {
			foldName = objectName + "/"
		}

		listResult, err := client.ListObjects(c, bucketName, sdktypes.ListObjectsOptions{ShowRemovedObject: false,
			Prefix: foldName})

		if err != nil {
			return toCmdErr(err)
		}

		if len(listResult.Objects) == 0 {
			return toCmdErr(errors.New("no objects to delete in the folder"))
		}

		// TODO use one txn to broadcast multi delete object messages
		for _, object := range listResult.Objects {
			// no need to return err if some objects failed
			deleteObjectAndWaitTxn(client, c, bucketName, object.ObjectInfo.ObjectName)
		}

	} else {
		err = deleteObjectAndWaitTxn(client, c, bucketName, objectName)
		if err != nil {
			return toCmdErr(err)
		}

		fmt.Printf("delete: gnfd:// %s/ %s\n", bucketName, objectName)
	}

	return nil
}

func deleteObjectAndWaitTxn(cli client.Client, c context.Context, bucketName, objectName string) error {
	txnHash, err := cli.DeleteObject(c, bucketName, objectName, sdktypes.DeleteObjectOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		fmt.Printf("failed to delele object %s err:%v\n", objectName, err)
		return err
	}

	err = waitTxnStatus(cli, c, txnHash, "DeleteObject")
	if err != nil {
		return err
	}

	fmt.Printf("deleted: %s\n", objectName)
	return nil
}

// deleteGroup send the deleteGroup msg to greenfield
func deleteGroup(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return toCmdErr(fmt.Errorf("args number more than one"))
	}
	groupName, err := getGroupNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx, false)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelDelGroup := context.WithCancel(globalContext)
	defer cancelDelGroup()

	txnHash, err := client.DeleteGroup(c, groupName, sdktypes.DeleteGroupOption{TxOpts: &TxnOptionWithSyncMode})
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txnHash, "DeleteGroup")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("delete group: %s successfully, txn hash: %s \n", groupName, txnHash)
	return nil
}
