package main

import (
	"cosmossdk.io/math"
	"fmt"
	storagetypes "github.com/bnb-chain/greenfield/x/storage/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

func cmdMirrorResource() *cli.Command {
	return &cli.Command{
		Name:      "mirror",
		Action:    Mirror,
		ArgsUsage: "",
		Description: `
Mirror resource to BSC

Examples:
# Mirror a bucket
$ gnfd-cmd -c config.toml mirror --resource bucket --id 1`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     resourceFlagName,
				Value:    "",
				Usage:    "resource type(group, bucket, object)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     IdFlagName,
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

	resource := ctx.String(resourceFlagName)
	id := math.NewUintFromString(ctx.String(IdFlagName))
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
