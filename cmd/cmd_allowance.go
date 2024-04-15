package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"cosmossdk.io/math"

	"github.com/bnb-chain/greenfield/sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

// cmdGrantAllowance grant allowance to a account
func cmdGrantAllowance() *cli.Command {
	return &cli.Command{
		Name:      "grant",
		Action:    grantAllowance,
		Usage:     "grant allowance",
		ArgsUsage: "",
		Description: `
The command is used to grant allowance to the grantee. --grantee defines the grantee address, --expire defines the expiration time in second

Examples:
$ gnfd-cmd fee grant --grantee 0x... --allowance 10000000000000000 --expire 3600`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  granteeFlag,
				Value: "",
				Usage: "the address hex string of the grantee",
			},
			&cli.StringFlag{
				Name:  allowanceFlag,
				Value: "",
				Usage: "the bnb in wei",
			},
			&cli.Uint64Flag{
				Name:  expireTimeFlag,
				Value: 0,
				Usage: "set the expire unix time stamp of the allowance",
			},
		},
	}
}

func grantAllowance(ctx *cli.Context) error {
	grantee := ctx.String(granteeFlag)
	granteeAddr, err := sdk.AccAddressFromHexUnsafe(grantee)
	if err != nil {
		return err
	}
	allowancesStr := ctx.String(allowanceFlag)
	allowance, ok := math.NewIntFromString(allowancesStr)
	if !ok {
		return toCmdErr(errors.New("convert string to int failed"))
	}
	var expireTime *time.Time
	if ctx.Uint64(expireTimeFlag) > 0 {
		temp := time.Unix(int64(ctx.Uint64(expireTimeFlag)), 0)
		expireTime = &temp
	}

	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}
	c, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()

	txHash, err := client.GrantBasicAllowance(c, granteeAddr.String(), allowance, expireTime, types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	err = waitTxnStatus(client, c, txHash, "GrantBasicAllowance")
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("Grant %s to %s succ, txHash=%s\n", allowance.String(), grantee, txHash)

	return nil
}
