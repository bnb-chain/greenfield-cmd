package main

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	paymenttypes "github.com/bnb-chain/greenfield/x/payment/types"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/urfave/cli/v2"
)

func cmdCreatePaymentAccount() *cli.Command {
	return &cli.Command{
		Name:      "payment-create-account",
		Action:    CreatePaymentAccount,
		ArgsUsage: "",
		Description: `
Create a payment account

Examples:
# Create a transfer
$ gnfd-cmd -c config.toml create-payment-account `,
	}
}

func CreatePaymentAccount(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}

	creator := km.GetAddr().String()
	msg := paymenttypes.NewMsgCreatePaymentAccount(creator)

	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}
	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("create-payment-account for %s failed, txHash=%s\n", creator, txHash))
	}
	fmt.Printf("create-payment-account for %s succ, txHash: %s\n", creator, txHash)
	return nil
}

// cmdPaymentDeposit
func cmdPaymentDeposit() *cli.Command {
	return &cli.Command{
		Name:   "payment-deposit",
		Action: Deposit,
		Usage:  "deposit",
		Description: `
Make a deposit into stream account 

Examples:
# deposit a stream account
$ gnfd-cmd -c config.toml payment-deposit --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     toAddressFlagName,
				Value:    "",
				Usage:    "the stream account",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
				Value: "",
				Usage: "the amount to be deposited",
			},
		},
	}
}

func Deposit(ctx *cli.Context) error {
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

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("invalid amount %s", amountStr))
	}
	msg := paymenttypes.NewMsgDeposit(
		km.GetAddr().String(),
		toAddr,
		amount,
	)
	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}

	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("Deposit %s to %s failed, txHash=%s\n", amount.String(), toAddr, txHash))
	}
	fmt.Printf("Deposit %s to %s succ, txHash=%s\n", amount.String(), toAddr, txHash)
	return nil
}

func cmdPaymentWithdraw() *cli.Command {
	return &cli.Command{
		Name:   "payment-withdraw",
		Action: Withdraw,
		Usage:  "withdraw",
		Description: `
Make a withdrawal from stream account 

Examples:
# withdraw from a stream account
$ gnfd-cmd -c config.toml payment-withdraw --toAddress 0x.. --amount 12345`,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     fromAddressFlagName,
				Value:    "",
				Usage:    "the stream account",
				Required: true,
			},
			&cli.StringFlag{
				Name:  amountFlagName,
				Value: "",
				Usage: "the amount to be withdrew",
			},
		},
	}
}

func Withdraw(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	fromAddr := ctx.String(fromAddressFlagName)
	amountStr := ctx.String(amountFlagName)
	amount, _ := math.NewIntFromString(amountStr)

	km, err := client.ChainClient.GetKeyManager()
	if err != nil {
		return toCmdErr(err)
	}

	amount, ok := math.NewIntFromString(amountStr)
	if !ok {
		return toCmdErr(fmt.Errorf("invalid amount %s", amountStr))
	}
	msg := paymenttypes.NewMsgWithdraw(
		km.GetAddr().String(),
		fromAddr,
		amount,
	)
	resp, err := client.ChainClient.BroadcastTx([]sdk.Msg{msg}, nil)
	if err != nil {
		return toCmdErr(err)
	}

	txHash := resp.TxResponse.TxHash
	if resp.TxResponse.Code != 0 {
		return toCmdErr(fmt.Errorf("Withdraw %s from %s failed, txHash=%s\n", amount.String(), fromAddr, txHash))
	}
	fmt.Printf("Withdraw %s from %s succ, txHash=%s\n", amount.String(), fromAddr, txHash)
	return nil
}

// cmdListPaymentAccounts list the payment accounts belong to the owner
func cmdListPaymentAccounts() *cli.Command {
	return &cli.Command{
		Name:      "ls-payment-account",
		Action:    listPaymentAccounts,
		Usage:     "list payment accounts of the user",
		ArgsUsage: "",
		Description: `
List payment accounts of the user.

Examples:
$ gnfd-cmd -c config.toml ls-payment-account `,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  userAddressFlagName,
				Value: "",
				Usage: "indicate user's payment accounts to be list, account address can be omitted for current user",
			},
		},
	}
}

func listPaymentAccounts(ctx *cli.Context) error {
	client, err := NewClient(ctx)
	if err != nil {
		return toCmdErr(err)
	}

	c, cancelCreateBucket := context.WithCancel(globalContext)
	defer cancelCreateBucket()

	var userAddr string
	userAddrStr := ctx.String(userAddressFlagName)
	if userAddrStr != "" {
		userAddr = userAddrStr
	} else {
		km, err := client.ChainClient.GetKeyManager()
		if err != nil {
			return toCmdErr(err)
		}
		userAddr = km.GetAddr().String()
	}

	req := paymenttypes.QueryGetPaymentAccountsByOwnerRequest{Owner: userAddr}

	accounts, err := client.ChainClient.PaymentQueryClient.GetPaymentAccountsByOwner(c, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			fmt.Println("Accounts not exist")
			return nil
		}
		return toCmdErr(err)
	}

	if len(accounts.PaymentAccounts) == 0 {
		fmt.Println("Accounts not exist")
		return nil
	}

	fmt.Println("payment accounts list:")
	for i, a := range accounts.PaymentAccounts {
		fmt.Printf("%d: %s \n", i+1, a)
	}
	return nil
}
