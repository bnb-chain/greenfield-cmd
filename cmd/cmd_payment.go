package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/bnb-chain/greenfield-go-sdk/client/gnfdclient"
	"github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/urfave/cli/v2"
)

// cmdGetQuotaPrice query the quota price of the specific sp
func cmdGetQuotaPrice() *cli.Command {
	return &cli.Command{
		Name:      "get-price",
		Action:    getQuotaPrice,
		Usage:     "get the quota price of sp",
		ArgsUsage: "",
		Description: `
Get the quota price of the specific sp, the command need to set the sp address with --spAddress

Examples:
$ gnfd  get-price --spAddress "0x.."`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     spAddressFlagName,
				Value:    "",
				Usage:    "indicate the storage provider chain address string",
				Required: true,
			},
		},
	}
}

// cmdBuyQuota buy the read quota of the bucket
func cmdBuyQuota() *cli.Command {
	return &cli.Command{
		Name:      "buy-quota",
		Action:    buyQuotaForBucket,
		Usage:     "update bucket meta on chain",
		ArgsUsage: "BUCKET-URL",
		Description: `
Update the visibility, payment account or read quota meta of the bucket

Examples:
$ gnfd  buy-quota  --chargedQuota 1000000  gnfd://bucket-name`,
		Flags: []cli.Flag{
			&cli.Uint64Flag{
				Name:     chargeQuotaFlagName,
				Value:    0,
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
$ gnfd  quota-info  gnfd://bucket-name`,
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

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	targetQuota := ctx.Uint64(chargeQuotaFlagName)
	if targetQuota == 0 {
		return toCmdErr(errors.New("target quota not set"))
	}

	broadcastMode := tx.BroadcastMode_BROADCAST_MODE_BLOCK
	txnOpt := types.TxOption{Mode: &broadcastMode}
	txnHash, err := client.BuyQuotaForBucket(c, bucketName, targetQuota, gnfdclient.BuyQuotaOption{TxOpts: &txnOpt})

	if err != nil {
		fmt.Println("buy quota error:", err.Error())
		return nil
	}

	fmt.Printf("buy quota for bucket: %s successfully, txn hash: %s\n", bucketName, txnHash)
	return nil
}

// getQuotaPrice query the quota price info of sp from greenfield chain
func getQuotaPrice(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	spAddressStr := ctx.String(spAddressFlagName)
	if spAddressStr == "" {
		return toCmdErr(errors.New("fail to fetch sp address"))
	}

	spAddr, err := sdk.AccAddressFromHexUnsafe(spAddressStr)
	if err != nil {
		return toCmdErr(err)
	}

	price, err := client.GetQuotaPrice(c, spAddr)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Println("get bucket read quota price:", price, " wei/byte")
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

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	quotaInfo, err := client.GetBucketReadQuota(c, bucketName)
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("quota info:\n charged quota:%d \nfree quota:%d \n consumed quota:%d \n",
		quotaInfo.ReadQuotaSize, quotaInfo.SPFreeReadQuotaSize, quotaInfo.ReadConsumedSize)
	return nil
}
