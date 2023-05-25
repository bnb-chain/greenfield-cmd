package main

import (
	"context"
	"errors"
	"fmt"

	sdktypes "github.com/bnb-chain/greenfield-go-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdBuyQuota buy the read quota of the bucket
func cmdBuyQuota() *cli.Command {
	return &cli.Command{
		Name:      "buy-quota",
		Action:    buyQuotaForBucket,
		Usage:     "update bucket quota info",
		ArgsUsage: "BUCKET-URL",
		Description: `
Update the read quota metadata of the bucket, indicating the target quota of the bucket.
The command need to set the target quota with --chargedQuota 

Examples:
$ gnfd-cmd payment buy-quota  --chargedQuota 1000000  gnfd://bucket-name`,
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:     chargeQuotaFlag,
				Usage:    "indicate the target quota to be set for the bucket",
				Required: true,
			},
		},
	}
}

func cmdGetQuotaInfo() *cli.Command {
	return &cli.Command{
		Name:      "quota-info",
		Action:    getQuotaInfo,
		Usage:     "get quota info of the bucket",
		ArgsUsage: "BUCKET-URL",
		Description: `
Get charged quota, free quota and consumed quota info from storage provider 

Examples:
$ gnfd -c config.toml payment quota-info  gnfd://bucket-name`,
	}
}

// buyQuotaForBucket set the charged quota meta of bucket on chain
func buyQuotaForBucket(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelBuyQuota := context.WithCancel(globalContext)
	defer cancelBuyQuota()

	// if bucket not exist, no need to buy quota
	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	targetQuota := ctx.Uint64(chargeQuotaFlag)
	if targetQuota == 0 {
		return toCmdErr(errors.New("target quota not set"))
	}

	txnHash, err := client.BuyQuotaForBucket(c, bucketName, targetQuota, sdktypes.BuyQuotaOption{TxOpts: &TxnOptionWithSyncMode})

	if err != nil {
		fmt.Println("buy quota error:", err.Error())
		return nil
	}

	fmt.Printf("buy quota for bucket: %s successfully, txn hash: %s\n", bucketName, txnHash)
	return nil
}

// getQuotaInfo query the quota price info of sp from greenfield chain
func getQuotaInfo(ctx *cli.Context) error {
	bucketName, err := getBucketNameByUrl(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelGetQuota := context.WithCancel(globalContext)
	defer cancelGetQuota()

	// if bucket not exist, no need to get info of quota
	_, err = client.HeadBucket(c, bucketName)
	if err != nil {
		return toCmdErr(ErrBucketNotExist)
	}

	quotaInfo, err := client.GetBucketReadQuota(c, bucketName)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("quota info:\n charged quota:%d \nfree quota:%d \n consumed quota:%d \n",
		quotaInfo.ReadQuotaSize, quotaInfo.SPFreeReadQuotaSize, quotaInfo.ReadConsumedSize)
	return nil
}
