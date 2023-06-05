package main

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/math"

	"cosmossdk.io/math"

	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/bnb-chain/greenfield/sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdCreateBucket create a new Bucket
func cmdCreateBucket() *cli.Command {
	return &cli.Command{
		Name:      "create",
		Action:    createBucket,
		Usage:     "create a new bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Create a new bucket and set a createBucketMsg to storage provider.
The bucket name should unique and the default visibility is private.
The command need to set the primary SP address with --primarySP.

Examples:
# Create a new bucket called gnfd-bucket, visibility is public-read
$ gnfd-cmd bucket create --visibility=public-read  gnfd://gnfd-bucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  primarySPFlag,
				Value: "",
				Usage: "indicate the primarySP address, using the string type",
			},
			&cli.StringFlag{
				Name:  paymentFlag,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlag,
				Value: 0,
				Usage: "indicate the read quota info of the bucket",
			},
			&cli.GenericFlag{
				Name: visibilityFlag,
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
		Name:      "update",
		Action:    updateBucket,
		Usage:     "update bucket meta on chain",
		ArgsUsage: "BUCKET-URL",
		Description: `
Update the visibility, payment account or read quota meta of the bucket.
The visibility value can be public-read, private or inherit.
You can update only one item or multiple items at the same time.

Examples:
update visibility and the payment address of the gnfd-bucket
$ gnfd-cmd bucket update --visibility=public-read --paymentAddress xx  gnfd://gnfd-bucket`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  paymentFlag,
				Value: "",
				Usage: "indicate the PaymentAddress info, using the string type",
			},
			&cli.Uint64Flag{
				Name:  chargeQuotaFlag,
				Usage: "indicate the read quota info of the bucket",
			},
			&cli.GenericFlag{
				Name: visibilityFlag,
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
		Name:      "ls",
		Action:    listBuckets,
		Usage:     "list buckets",
		ArgsUsage: "",
		Description: `
List the bucket names and bucket ids of the user.

Examples:
$ gnfd-cmd bucket ls`,
	}
}

func cmdMirrorBucket() *cli.Command {
	return &cli.Command{
		Name:      "mirror",
		Action:    mirrorBucket,
		Usage:     "mirror bucket to BSC",
		ArgsUsage: "",
		Description: `
Mirror a bucket as NFT to BSC

Examples:
# Mirror a bucket using bucket id
$ gnfd-cmd bucket mirror --id 1

# Mirror a bucket using bucket name
$ gnfd-cmd bucket mirror --name yourBucketName
`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     IdFlag,
				Value:    "",
				Usage:    "bucket id",
				Required: false,
			},
			&cli.StringFlag{
				Name:     bucketNameFlag,
				Value:    "",
				Usage:    "bucket name",
				Required: false,
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

	primarySpAddrStr := ctx.String(primarySPFlag)
	if primarySpAddrStr == "" {
		// if primarySP not set, choose sp0 as the primary sp
		spInfo, err := client.ListStorageProviders(c, false)
		if err != nil {
			return toCmdErr(errors.New("fail to get primary sp address"))
		}
		primarySpAddrStr = spInfo[0].GetOperatorAddress()
		fmt.Println("choose primary sp:", spInfo[0].GetEndpoint())
	}

	opts := sdktypes.CreateBucketOptions{}
	paymentAddrStr := ctx.String(paymentFlag)
	if paymentAddrStr != "" {
		opts.PaymentAddress = paymentAddrStr
	}

	visibility := ctx.Generic(visibilityFlag)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlag)
	if chargedQuota > 0 {
		opts.ChargedQuota = chargedQuota
	}

	opts.TxOpts = &types.TxOption{Mode: &SyncBroadcastMode}
	txnHash, err := client.CreateBucket(c, bucketName, primarySpAddrStr, opts)
	if err != nil {
		return toCmdErr(err)
	}

	err = waitTxnStatus(client, c, txnHash, "CreateBucket")
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

	c, cancelUpdateBucket := context.WithCancel(globalContext)
	defer cancelUpdateBucket()

	// if bucket not exist, no need to update it
	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	opts := sdktypes.UpdateBucketOptions{}
	paymentAddrStr := ctx.String(paymentFlag)
	if paymentAddrStr != "" {
		opts.PaymentAddress = paymentAddrStr
	}

	visibility := ctx.Generic(visibilityFlag)
	if visibility != "" {
		visibilityTypeVal, typeErr := getVisibilityType(fmt.Sprintf("%s", visibility))
		if typeErr != nil {
			return typeErr
		}
		opts.Visibility = visibilityTypeVal
	}

	chargedQuota := ctx.Uint64(chargeQuotaFlag)
	if chargedQuota > 0 {
		opts.ChargedQuota = &chargedQuota
	}

	opts.TxOpts = &TxnOptionWithSyncMode

	txnHash, err := client.UpdateBucketInfo(c, bucketName, opts)
	if err != nil {
		fmt.Println("update bucket error:", err.Error())
		return nil
	}

	err = waitTxnStatus(client, c, txnHash, "UpdateBucket")
	if err != nil {
		return toCmdErr(err)
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

	bucketListRes, err := client.ListBuckets(c)

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
		if !bucket.Removed {
			fmt.Printf("bucket name: %s, bucket id: %s \n", info.BucketName, info.Id)
		}
	}
	return nil

}

func mirrorBucket(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	id := math.NewUintFromString(ctx.String(IdFlag))
	bucketName := ctx.String(bucketNameFlag)

	c, cancelContext := context.WithCancel(globalContext)
	defer cancelContext()

	txResp, err := client.MirrorBucket(c, id, bucketName, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("mirror bucket succ, txHash: %s\n", txResp.TxHash)
	return nil
}
