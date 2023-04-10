package main

import (
	"context"
	"fmt"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	"github.com/bnb-chain/greenfield-go-sdk/client/sp"
	"github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdCreateBucket create a new Bucket
func cmdCreateBucket() *cli.Command {
	return &cli.Command{
		Name:      "mb",
		Action:    createBucket,
		Usage:     "create bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Create a new bucket and set a createBucketMsg to storage provider.
The bucket name should unique and the default visibility is private.
The command need to set the primary SP address with --primarySP

Examples:
# Create a new bucket called gnfdBucket, visibility is public-read
$ gnfd-cmd -c config.toml mb  --primarySP  --visibility=public-read  gnfd://gnfdBucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     primarySPFlagName,
				Value:    "",
				Usage:    "indicate the primarySP address, using the string type",
				Required: true,
			},
			&cli.StringFlag{
				Name:  paymentFlagName,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlagName,
				Value: 0,
				Usage: "indicate the read quota info of the bucket",
			},
			&cli.GenericFlag{
				Name: visibilityFlagName,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: privateType,
				},
				Usage: "set visibility of the bucket",
			},
		},
	}
}

// cmdUpdateBucket create a new Bucket
func cmdUpdateBucket() *cli.Command {
	return &cli.Command{
		Name:      "update-bucket",
		Action:    updateBucket,
		Usage:     "update bucket meta on chain",
		ArgsUsage: "BUCKET-URL",
		Description: `
Update the visibility, payment account or read quota meta of the bucket.
The visibility value can be public-read, private or inherit.
You can update only one item or multiple items at the same time.

Examples:
update visibility and the payment address of the gnfdBucket
$ gnfd-cmd -c config.toml update-bucket --visibility=public-read --paymentAddress xx  gnfd://gnfdBucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  paymentFlagName,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlagName,
				Value: 0,
				Usage: "indicate the read quota info of the bucket",
			},
			&cli.GenericFlag{
				Name: visibilityFlagName,
				Value: &CmdEnumValue{
					Enum:    []string{publicReadType, privateType, inheritType},
					Default: privateType,
				},
				Usage: "set visibility of the bucket",
			},
		},
	}
}

// cmdListBuckets list the bucket of the owner
func cmdListBuckets() *cli.Command {
	return &cli.Command{
		Name:      "ls-bucket",
		Action:    listBuckets,
		Usage:     "list buckets of the user",
		ArgsUsage: "",
		Description: `
List the bucket names and bucket ids of the user.

Examples:
$ gnfd-cmd -c config.toml ls-bucket `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  userAddressFlagName,
				Value: "",
				Usage: "indicate which user's buckets to be list, you" +
					" don't need to specify this if you want to list your own bucket ",
			},
		},
	}
}

// createBucket send the create bucket request to storage provider
func createBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	primarySpAddrStr := ctx.String(primarySPFlagName)
	primarySpAddr := sdk.MustAccAddressFromHex(primarySpAddrStr)

	paymentAddrStr := ctx.String(paymentFlagName)
	opts := gnfdclient.CreateBucketOptions{}
	if paymentAddrStr != "" {
		opts.PaymentAddress = sdk.MustAccAddressFromHex(paymentAddrStr)
	}

	visibility := ctx.Generic(visibilityFlagName)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlagName)
	if chargedQuota > 0 {
		opts.ChargedQuota = chargedQuota
	}

	txnHash, err := client.CreateBucket(c, bucketName, primarySpAddr, opts)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("create bucket %s succ, txn hash: %s\n", bucketName, txnHash)
	return nil
}

// updateBucket send the create bucket request to storage provider
func updateBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	opts := gnfdclient.UpdateBucketOption{}
	paymentAddrStr := ctx.String(paymentFlagName)
	if paymentAddrStr != "" {
		paymentAddress, err := sdk.AccAddressFromHexUnsafe(paymentAddrStr)
		if err != nil {
			return err
		}
		opts.PaymentAddress = paymentAddress
	}

	visibility := ctx.Generic(visibilityFlagName)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlagName)
	if chargedQuota > 0 {
		opts.ChargedQuota = &chargedQuota
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	opts.TxOpts = &txnOpt

	_, err = client.UpdateBucketInfo(c, bucketName, opts)
	if err != nil {
		fmt.Println("update bucket error:", err.Error())
		return nil
	}

	bucketInfo, err := client.HeadBucket(c, bucketName)
	if err != nil {
		// head fail, no need to print the error
		return nil
	}

	fmt.Printf("latest bucket meta on chain:\nvisibility:%s\nread quota:%d\npayment address:%s \n", bucketInfo.GetVisibility().String(),
		bucketInfo.GetChargedReadQuota(), bucketInfo.GetPaymentAddress())
	return nil
}

// listBuckets list the buckets of the specific owner
func listBuckets(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	var userAddr string
	userAddrStr := ctx.String(userAddressFlagName)
	if userAddrStr != "" {
		userAddr = userAddrStr
	} else {
		km, err := client.ChainClient.GetKeyManager()
		if err != nil {
			return toCmdErr(err)
		}
		userAddr = km.GetAddr().String()
	}

	bucketListRes, err := client.SPClient.ListBuckets(c, sp.UserInfo{Address: userAddr},
		sp.NewAuthInfo(false, ""))

	if err != nil {
		return toCmdErr(err)
	}

	if len(bucketListRes.Buckets) == 0 {
		fmt.Println("no buckets")
		return nil
	}

	fmt.Println("bucket list:")
	for _, bucket := range bucketListRes.Buckets {
		info := bucket.BucketInfo
		fmt.Printf("bucket name: %s, bucket id: %s \n", info.BucketName, info.Id)
	}
	return nil

}
