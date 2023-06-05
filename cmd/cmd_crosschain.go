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

	err = waitTxnStatus(client, c, txResp.TxHash, "TransferOut")
	if err != nil {
		return toCmdErr(err)
	}

	fmt.Printf("transfer out %s wei to %s succ, txHash: %s\n", amountStr, toAddr, txResp.TxHash)
	return nil
}
