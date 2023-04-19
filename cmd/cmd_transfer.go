package main

import (
	"cosmossdk.io/math"
	"fmt"
	gnfdsdktypes "github.com/bnb-chain/greenfield/sdk/types"
	bridgetypes "github.com/bnb-chain/greenfield/x/bridge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/urfave/cli/v2"
)

func cmdTransfer() *cli.Command {
	return &cli.Command{
		Name:      "transfer",
		Action:    Transfer,
		Usage:     "transfer",
		ArgsUsage: "",
		Description: `
Make a transfer from your account to a dest account

Examples:
# Create a transfer
$ gnfd-cmd -c config.toml transfer --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlagName,
				Value:    "",
				Usage:    "the receiver address in BSC",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
				Value: "",
				Usage: "the amount to be sent",
			},
		},
	}
}

func Transfer(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	toAddr := ctx.String(toAddressFlagName)
	amountStr := ctx.String(amountFlagName)
	amount, _ := math.NewIntFromString(amountStr)

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}
	to, err := sdk.AccAddressFromHexUnsafe(toAddr)
	if err != nil {
		return toCmdErr(err)
	}

	msg := banktypes.NewMsgSend(
		km.GetAddr(),
		to,
		sdk.NewCoins(sdk.NewCoin(gnfdsdktypes.Denom, amount)),
	)

	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}
	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("transfer to %s failed, txHash=%s\n", txHash))

	}
	fmt.Printf("transfer to %s succ, txHash: %s\n", toAddr, txHash)
	return nil
}

// cmdTransferOut makes a transfer from Greenfield to BSC
func cmdTransferOut() *cli.Command {
	return &cli.Command{
		Name:      "transfer-out",
		Action:    TransferOut,
		Usage:     "transfer out",
		ArgsUsage: "",
		Description: `
Create a transfer from Greenfield to BSC

Examples:
# Create a transfer
$ gnfd-cmd -c config.toml transfer-out --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlagName,
				Value:    "",
				Usage:    "the receiver address in BSC",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
				Value: "",
				Usage: "the amount to be sent",
			},
		},
	}
}

func TransferOut(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	toAddr := ctx.String(toAddressFlagName)
	amountStr := ctx.String(amountFlagName)
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
		return toCmdErr(fmt.Errorf("transfer out to %s failed, txHash=%s\n", toAddr, txHash))
	}
	fmt.Printf("transfer out to %s succ, txHash: %s\n", toAddr, txHash)
	return nil
}
