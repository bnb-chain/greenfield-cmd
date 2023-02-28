package main

import (
	"fmt"

	"github.com/bnb-chain/greenfield/sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdDelBucket delete an existing Bucket,the bucket must be empty
func cmdDelBucket() *cli.Command {
	return &cli.Command{
		Name:      "del-bucket",
		Action:    deleteBucket,
		Usage:     "delete an existing bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Send a deleteBucket txn to greenfield chain, the bucket must be empty before deleting

Examples:
# Del an existing bucket
$ gnfd-cmd  del-bucket gnfd://bucketname`,
	}
}

// cmdDelObject delete an existing object in bucket
func cmdDelObject() *cli.Command {
	return &cli.Command{
		Name:      "del-obj",
		Action:    deleteObject,
		Usage:     "delete an existing object",
		ArgsUsage: "BUCKET-URL",
		Description: `
Send a deleteObject txn to greenfield chain

Examples:
# Del an existing object
$ gnfd-cmd del-obj gnfd://bucketname/objectname`,
	}
}

// deleteBucket send the deleteBucket msg to greenfield
func deleteBucket(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("the args number should be one")
	}
	bucketName, err := getBucketName(ctx)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	gnfdResp := client.DelBucket(bucketName, types.TxOption{Mode: &broadcastMode})
	if gnfdResp.Err != nil {
		fmt.Println("delete bucket error:", gnfdResp.Err.Error())
		return err
	}

	fmt.Println("delete bucket finish, txn hash:", gnfdResp.TxnHash)
	return nil
}

// deleteObject send the deleteBucket msg to greenfield
func deleteObject(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("the args number should be one")
	}

	urlInfo := ctx.Args().Get(0)
	bucketName, objectName, err := getObjAndBucketNames(urlInfo)
	if err != nil {
		return nil
	}

	client, err := NewClient(ctx)
	if err != nil {
		return err
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	gnfdResp := client.DelObject(bucketName, objectName, types.TxOption{Mode: &broadcastMode})
	if gnfdResp.Err != nil {
		fmt.Println("delete object error:", gnfdResp.Err.Error())
		return err
	}

	fmt.Println("delete object finish, txn hash:", gnfdResp.TxnHash)
	return nil
}
