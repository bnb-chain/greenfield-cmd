package main

import (
	"context"
	"fmt"

	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdDelBucket delete an existed Bucket, the bucket must be empty
func cmdDelBucket() *cli.Command {
	return &cli.Command{
		Name:      "del-bucket",
		Action:    deleteBucket,
		Usage:     "delete an existed bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Send a deleteBucket txn to greenfield chain, the bucket must be empty before deleting

Examples:
# Delete an existed bucket called gnfdBucket
$ gnfd-cmd -c config.toml del-bucket gnfd://gnfdBucket/gnfdObject`,
	}
}

// cmdDelObject delete an existed object in bucket
func cmdDelObject() *cli.Command {
	return &cli.Command{
		Name:      "del-obj",
		Action:    deleteObject,
		Usage:     "delete an existed object",
		ArgsUsage: "BUCKET-URL",
		Description: `
Send a deleteObject txn to greenfield chain

Examples:
# Delete an existed object called gnfdObject
$ gnfd-cmd -c config.toml del-obj gnfd://gnfdBucket/gnfdObject`,
	}
}

// cmdDelGroup delete an existed group
func cmdDelGroup() *cli.Command {
	return &cli.Command{
		Name:      "del-group",
		Action:    deleteGroup,
		Usage:     "delete an existed group",
		ArgsUsage: "GROUP-URL",
		Description: `
Send a deleteGroup txn to greenfield chain

Examples:
# Delete an existed group
$ gnfd-cmd -c config.toml del-group gnfd://group-name`,
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

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelDelBucket := context.WithCancel(globalContext)
	defer cancelDelBucket()

	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	txnHash, err := client.DeleteBucket(c, bucketName, sdktypes.DeleteBucketOption{TxOpts: &txnOpt})
	if err != nil {
		fmt.Println("delete bucket error:", err.Error())
		return nil
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

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelDelObject := context.WithCancel(globalContext)
	defer cancelDelObject()

	_, err = client.HeadObject(c, bucketName, objectName)
	if err != nil {
		return toCmdErr(ErrObjectNotExist)
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	txnHash, err := client.DeleteObject(c, bucketName, objectName, sdktypes.DeleteObjectOption{TxOpts: &txnOpt})
	if err != nil {
		fmt.Println("delete object error:", err.Error())
		return err
	}

	fmt.Printf("delete object %s successfully, txn hash:%s \n",
		objectName, txnHash)
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

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelDelGroup := context.WithCancel(globalContext)
	defer cancelDelGroup()

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	txnHash, err := client.DeleteGroup(c, groupName, sdktypes.DeleteGroupOption{TxOpts: &txnOpt})
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("delete group: %s successfully, txn hash: %s \n", groupName, txnHash)
	return nil
}
