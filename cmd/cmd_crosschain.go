package main

import (
	"fmt"

	"cosmossdk.io/math"
	gnfdsdktypes "github.com/bnb-chain/greenfield/sdk/types"
	bridgetypes "github.com/bnb-chain/greenfield/x/bridge/types"
	storagetypes "github.com/bnb-chain/greenfield/x/storage/types"

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
$ gnfd-cmd -c config.toml transfer-out --toAddress 0x.. --amount 12345`,
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
				Usage:    "the amount of BNB to be sent",
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

	toAddr := ctx.String(toAddressFlag)
	amountStr := ctx.String(amountFlag)
	amount, _ := math.NewIntFromString(amountStr)

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}

	msgTransferOut := bridgetypes.NewMsgTransferOut(
		km.GetAddr().String(),
		toAddr,
		&sdk.Coin{Denom: gnfdsdktypes.Denom, Amount: amount},
	)

	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msgTransferOut}, nil)
	if err != nil {
		return toCmdErr(err)
	}

	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("transfer out %s BNB to %s failed, txHash=%s\n", amountStr, toAddr, txHash))
	}
	fmt.Printf("transfer out %s BNB to %s succ, txHash: %s\n", amountStr, toAddr, txHash)
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
$ gnfd-cmd -c config.toml mirror --resource bucket --id 1`,
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

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}
	addr := km.GetAddr()

	resource := ctx.String(resourceFlag)
	id := math.NewUintFromString(ctx.String(IdFlag))
	var msg sdk.Msg

	if resource == "group" {
		msg = storagetypes.NewMsgMirrorGroup(addr, id)
	} else if resource == "bucket" {
		msg = storagetypes.NewMsgMirrorBucket(addr, id)
	} else if resource == "object" {
		msg = storagetypes.NewMsgMirrorObject(addr, id)
	} else {
		return toCmdErr(fmt.Errorf("wrong resource type %s, expect one of (group, bucket, object)", resource))
	}

	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}
	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("mirror %s with id %s failed, txHash=%s\n", resource, id.String(), txHash))
	}
	fmt.Printf("mirror %s with id %s succ, txHash: %s\n", resource, id.String(), txHash)
	return nil
}
