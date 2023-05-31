package main

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	gnfdsdktypes "github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdTransferOut makes a transfer from Greenfield to BSC
func cmdTransferOut() *cli.Command {
	return &cli.Command{
		Name:      "transfer-out",
		Action:    TransferOut,
		Usage:     "transfer from greenfield to a BSC account",
		ArgsUsage: "",
		Description: `
Create a cross chain transfer from Greenfield to a BSC account

Examples:
# Create a crosschain transfer
$ gnfd-cmd crosschain transfer-out --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlag,
				Value:    "",
				Usage:    "the receiver address in BSC",
				Required: true,
			},
			&cli.StringFlag{
				Name:     amountFlag,
				Value:    "",
				Usage:    "the amount of wei to be sent",
				Required: true,
			},
		},
	}
}

func TransferOut(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, transfer := context.WithCancel(globalContext)
	defer transfer()

	toAddr := ctx.String(toAddressFlag)
	_, err = sdk.AccAddressFromHexUnsafe(toAddr)
	if err != nil {
		return toCmdErr(err)
	}
	amountStr := ctx.String(amountFlag)
	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("%s is not valid amount", amount))
	}
	txResp, err := client.TransferOut(c, toAddr, amount, gnfdsdktypes.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}

	txnHash := txResp.TxHash
	ctxTimeout, cancel := context.WithTimeout(context.Background(), ContextTimeout)
	defer cancel()

	txnResponse, err := client.WaitForTx(ctxTimeout, txnHash)
	if err != nil {
		return toCmdErr(fmt.Errorf("the txn: %s ,has been submitted, please check it later:%v", txnHash, err))
	}
	if txnResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("the TransferOut txn: %s has failed with response code: %d", txnHash, txnResponse.Code))
	}

	fmt.Printf("transfer out %s wei to %s succ, txHash: %s\n", amountStr, toAddr, txResp.TxHash)
	return nil
}

func cmdMirrorResource() *cli.Command {
	return &cli.Command{
		Name:      "mirror",
		Action:    Mirror,
		Usage:     "mirror resource to bsc",
		ArgsUsage: "",
		Description: `
Mirror resource to BSC

Examples:
# Mirror a bucket
$ gnfd-cmd crosschain mirror --resource bucket --id 1`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     resourceFlag,
				Value:    "",
				Usage:    "resource type(group, bucket, object)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     IdFlag,
				Value:    "",
				Usage:    "resource id",
				Required: true,
			},
		},
	}
}

func Mirror(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}
	resource := ctx.String(resourceFlag)
	id := math.NewUintFromString(ctx.String(IdFlag))
	groupName := ctx.String(groupNameFlag)
	bucketName := ctx.String(bucketNameFlag)
	objectName := ctx.String(objectNameFlag)

	c, cancelContext := context.WithCancel(globalContext)
	defer cancelContext()

	var txResp *sdk.TxResponse
	if resource == "group" {
		txResp, err = client.MirrorGroup(c, id, groupName, gnfdsdktypes.TxOption{})
	} else if resource == "bucket" {
		txResp, err = client.MirrorBucket(c, id, bucketName, gnfdsdktypes.TxOption{})
	} else if resource == "object" {
		txResp, err = client.MirrorObject(c, id, bucketName, objectName, gnfdsdktypes.TxOption{})
	} else {
		return toCmdErr(fmt.Errorf("wrong resource type %s, expect one of (group, bucket, object)", resource))
	}
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("mirror %s with id %s succ, txHash: %s\n", resource, id.String(), txResp.TxHash)
	return nil
}
