package main

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	spType "github.com/bnb-chain/greenfield/x/sp/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdMakeBucket create a new Bucket
func cmdCreateBucket() *cli.Command {
	return &cli.Command{
		Name:      "mb",
		Action:    createBucket,
		Usage:     "create bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Create a new bucket and set a createBucketMsg to storage provider, 
the bucket name should unique and the default acl is not public.

Examples:
# Create a new bucket
$ gnfd mb  gnfd://bucketname`,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "public",
				Value: false,
				Usage: "indicate whether the bucket is public",
			},
			&cli.StringFlag{
				Name:  "primarySP",
				Value: "",
				Usage: "indicate the primarySP address, using the string type",
			},
			&cli.StringFlag{
				Name:  "PaymentAddr",
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
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

	client, err := NewClient(ctx)
	if err != nil {
		log.Println("create client fail", err.Error())
		return err
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	isPublic := ctx.Bool("public")
	primarySpAddrStr := ctx.String("primarySP")
	paymentAddrStr := ctx.String("PaymentAddr")

	opts := gnfdclient.CreateBucketOptions{}
	opts.IsPublic = isPublic
	if paymentAddrStr != "" {
		opts.PaymentAddress = sdk.MustAccAddressFromHex(paymentAddrStr)
	}

	request := &spType.QueryStorageProvidersRequest{}
	chainCtx := context.Background()
	gnfdRep, err := client.ChainClient.StorageProviders(chainCtx, request)
	if err != nil {
		return err
	}

	if primarySpAddrStr == "" {
		spList := gnfdRep.GetSps()
		existPrimarySp := false
		for _, sp := range spList {
			if sp.Description.Moniker == "sp0" {
				existPrimarySp = true
				primarySpAddrStr = sp.GetOperatorAddress()
				if sp.Status.String() != "STATUS_IN_SERVICE" {
					return errors.New("primary sp")
				}
			}
		}

		if !existPrimarySp {
			return errors.New("not exist primary sp")
		}

	}

	primarySpAddr := sdk.MustAccAddressFromHex(primarySpAddrStr)
	gnfdResp := client.CreateBucket(c, bucketName, primarySpAddr, opts)
	if gnfdResp.Err != nil {
		return gnfdResp.Err
	}

	fmt.Println("create bucket succ, txn hash:", gnfdResp.TxnHash)
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
