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

Flags:
The command includes three flags that users can set when using the command to specify relevant values:
--grantee (granteeFlag)
	Type: String (cli.StringFlag)
	Default Value: Empty string
	Usage: Specifies the address of the grantee (in hexadecimal string format).
--allowance (allowanceFlag)
	Type: String
	Default Value: Empty string
	Usage: Specifies the amount of allowance in wei.
--expire (expireTimeFlag)
	Type: Unsigned 64-bit integer (cli.Uint64Flag)
	Default Value: 0
	Usage: Sets the expiration time of the allowance in UNIX timestamp (seconds).

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

// cmdRevokeAllowance revoke allowance from a account
func cmdRevokeAllowance() *cli.Command {
	return &cli.Command{
		Name:      "revoke",
		Action:    revokeAllowance,
		Usage:     "revoke allowance",
		ArgsUsage: "",
		Description: `
The command is used to revoke allowance from the grantee. --grantee defines the grantee address

Flags:
The command includes one flag that users can set when using the command to specify relevant values:
--grantee (granteeFlag)
	Type: String (cli.StringFlag)
	Default Value: Empty string
	Usage: Specifies the address of the grantee (in hexadecimal string format).

Examples:
$ gnfd-cmd fee revoke --grantee 0x...`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  granteeFlag,
				Value: "",
				Usage: "the address hex string of the grantee",
			},
		},
	}
}

func revokeAllowance(ctx *cli.Context) error {
	grantee := ctx.String(granteeFlag)
	granteeAddr, err := sdk.AccAddressFromHexUnsafe(grantee)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx, ClientOptions{IsQueryCmd: false})
	if err != nil {
		return toCmdErr(err)
	}
	c, cancelSetTag := context.WithCancel(globalContext)
	defer cancelSetTag()

	txHash, err := client.RevokeAllowance(c, granteeAddr.String(), types.TxOption{})
	if err != nil {
		return toCmdErr(err)
	}
	err = waitTxnStatus(client, c, txHash, "RevokeAllowance")
	if err != nil {
		return toCmdErr(err)
	}
	fmt.Printf("Revoke allowance from %s succ, txHash=%s\n", grantee, txHash)

	return nil
}
